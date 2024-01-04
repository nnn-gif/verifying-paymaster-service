package api

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ququzone/verifying-paymaster-service/config"
	"github.com/ququzone/verifying-paymaster-service/container"
	"github.com/ququzone/verifying-paymaster-service/contracts"
	"github.com/ququzone/verifying-paymaster-service/logger"
	"github.com/ququzone/verifying-paymaster-service/models"
	"github.com/ququzone/verifying-paymaster-service/types"
	"github.com/ququzone/verifying-paymaster-service/utils"
)

var (
	// one day
	validTimeDelay = new(big.Int).SetInt64(86400)
	uint48Ty, _    = abi.NewType("uint256", "uint48", []abi.ArgumentMarshaling{})
	timeRangeABI   = abi.Arguments{
		{Name: "validUntil", Type: uint48Ty},
		{Name: "validAfter", Type: uint48Ty},
	}
	emptySignature = make([]byte, 65)
)

type revertError struct {
	reason string // revert reason hex encoded
}

func (e *revertError) Error() string {
	return "execution reverted"
}

func (e *revertError) ErrorData() interface{} {
	return e.reason
}

type GasRemain struct {
	Remain      string `json:"remain"`
	LastRequest int64  `json:"last_request"`
	Used        string `json:"total_used"`
}

type PaymasterConfig struct {
	MaxGas      string `json:"max_gas"`
	VipContract string `json:"vip_contract"`
	MaxVipGas   string `json:"max_vip_gas"`
}

type Signer struct {
	Container   container.Container
	Client      *ethclient.Client
	Contract    common.Address
	Paymaster   *contracts.VerifyingPaymaster
	PrivateKey  *ecdsa.PrivateKey
	CreateGas   *big.Int
	MaxGas      *big.Int
	MaxVipGas   *big.Int
	VipContract *contracts.VipNFT
}

func NewSigner(con container.Container) (*Signer, error) {
	conf := config.Config()
	keyBytes, err := hex.DecodeString(conf.PrivateKey)
	if err != nil {
		return nil, err
	}
	privKey, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return nil, err
	}
	logger.S().Infof("VerifyingPaymaster contract: %s", conf.Contract)

	rpc, err := ethclient.Dial(conf.RPC)
	if err != nil {
		return nil, err
	}

	contract := common.HexToAddress(conf.Contract)
	paymaster, err := contracts.NewVerifyingPaymaster(contract, rpc)
	if err != nil {
		return nil, err
	}
	createGas, _ := new(big.Int).SetString(conf.CreateGas, 10)
	maxGas, _ := new(big.Int).SetString(conf.MaxGas, 10)

	vipContract, err := contracts.NewVipNFT(common.HexToAddress(conf.VipContract), rpc)
	if err != nil {
		return nil, err
	}
	maxVipGas, _ := new(big.Int).SetString(conf.VipMaxGas, 10)

	return &Signer{
		Container:   con,
		Client:      rpc,
		Contract:    contract,
		Paymaster:   paymaster,
		PrivateKey:  privKey,
		CreateGas:   createGas,
		MaxGas:      maxGas,
		VipContract: vipContract,
		MaxVipGas:   maxVipGas,
	}, nil
}

type PaymasterResult struct {
	PaymasterAndData     string `json:"paymasterAndData"`
	PreVerificationGas   string `json:"preVerificationGas"`
	VerificationGasLimit string `json:"verificationGasLimit"`
	CallGasLimit         string `json:"callGasLimit"`
}

func (s *Signer) Pm_sponsorUserOperation(op map[string]any, entryPoint string) (*PaymasterResult, error) {
	entryPoint = "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"
	userOp, err := types.NewUserOperation(op)
	if err != nil {
		return nil, err
	}

	account, err := (&models.Account{}).FindByAddress(s.Container.GetRepository(), strings.ToLower(userOp.Sender.String()))
	if nil != err || account == nil {
		return nil, errors.New("insufficient gas")
		// remove auto claim gas
		// account = &models.Account{
		// 	Address:     strings.ToLower(strings.ToLower(userOp.Sender.String())),
		// 	Enable:      true,
		// 	UsedGas:     "0",
		// 	RemainGas:   s.MaxGas.String(),
		// 	LastRequest: time.Now(),
		// }
		// err = s.Container.GetRepository().Save(account).Error
		// if nil != err {
		// 	logger.S().Errorf("save account error: %v", err)
		// 	return nil, err
		// }
	}

	// tempOp, _ := types.NewUserOperation(op)
	// preVerificationGas, verificationGas, callGas, err := estimate(
	// 	s.Client,
	// 	s.PrivateKey,
	// 	s.Contract,
	// 	s.Paymaster,
	// 	common.HexToAddress(entryPoint),
	// 	tempOp,
	// )
	// if err != nil {
	// 	return nil, err
	// }

	preVerificationGas, verificationGas, callGas, err := big.NewInt(52304), big.NewInt(100000), big.NewInt(33100), nil

	remainGas, _ := new(big.Int).SetString(account.RemainGas, 10)
	totalGas := new(big.Int).Add(preVerificationGas, verificationGas)
	totalGas = new(big.Int).Add(totalGas, callGas)
	totalGas = new(big.Int).Mul(totalGas, userOp.MaxFeePerGas)
	// Auto claim gas
	// if totalGas.Cmp(remainGas) > 0 {
	// 	if account.LastRequest.Unix()+86400 < time.Now().Unix() {
	// 		account.LastRequest = time.Now()
	// 		account.RemainGas = s.MaxGas.String()
	// 		err = s.Container.GetRepository().Save(account).Error
	// 		if nil != err {
	// 			logger.S().Errorf("save account error: %v", err)
	// 			return nil, err
	// 		}
	// 	}
	// }
	// remainGas, _ = new(big.Int).SetString(account.RemainGas, 10)
	if totalGas.Cmp(remainGas) > 0 {
		return nil, errors.New("insufficient gas")
	}
	usedGas, _ := new(big.Int).SetString(account.UsedGas, 10)
	account.UsedGas = new(big.Int).Add(usedGas, totalGas).String()
	account.RemainGas = new(big.Int).Sub(remainGas, totalGas).String()
	err = s.Container.GetRepository().Save(account).Error
	if nil != err {
		logger.S().Errorf("save account error: %v", err)
		return nil, err
	}

	// TODO: verify op rules:
	//  1. normal gas
	//  2. only for create
	validAfter := new(big.Int).SetInt64(time.Now().Unix())
	validUntil := new(big.Int).Add(validAfter, validTimeDelay)
	timeRangeData, err := timeRangeABI.Pack(validUntil, validAfter)
	if err != nil {
		return nil, err
	}
	userOp.PaymasterAndData = append(append(s.Contract.Bytes(), timeRangeData...), emptySignature...)
	userOp.Signature = []byte{}

	hash, err := s.Paymaster.GetHash(nil, contracts.UserOperation{
		Sender:               userOp.Sender,
		Nonce:                userOp.Nonce,
		InitCode:             userOp.InitCode,
		CallData:             userOp.CallData,
		CallGasLimit:         callGas,
		VerificationGasLimit: verificationGas,
		PreVerificationGas:   preVerificationGas,
		MaxFeePerGas:         userOp.MaxFeePerGas,
		MaxPriorityFeePerGas: userOp.MaxPriorityFeePerGas,
		PaymasterAndData:     userOp.PaymasterAndData,
		Signature:            userOp.Signature,
	}, validUntil, validAfter)
	if err != nil {
		return nil, err
	}
	signature, err := utils.SignMessage(s.PrivateKey, hash[:])
	if err != nil {
		return nil, err
	}

	// TODO: set gas
	return &PaymasterResult{
		PaymasterAndData:     hexutil.Encode(append(append(s.Contract.Bytes(), timeRangeData...), signature...)),
		PreVerificationGas:   hexutil.Encode(preVerificationGas.Bytes()),
		VerificationGasLimit: hexutil.Encode(verificationGas.Bytes()),
		CallGasLimit:         hexutil.Encode(callGas.Bytes()),
	}, nil
}

func (s *Signer) Pm_gasRemain(addr string) (*GasRemain, error) {
	account, err := (&models.Account{}).FindByAddress(s.Container.GetRepository(), strings.ToLower(addr))
	if nil != err {
		logger.S().Errorf("Query account error: %v", err)
		return nil, err
	}
	if account == nil || !account.Enable {
		return &GasRemain{
			Remain:      "0",
			Used:        "0",
			LastRequest: 0,
		}, nil
	}
	return &GasRemain{
		Remain:      account.RemainGas,
		Used:        account.UsedGas,
		LastRequest: account.LastRequest.Unix(),
	}, nil
}

func (s *Signer) Pm_config() (*PaymasterConfig, error) {
	return &PaymasterConfig{
		MaxGas:      config.Config().MaxGas,
		VipContract: config.Config().VipContract,
		MaxVipGas:   config.Config().VipMaxGas,
	}, nil
}

func (s *Signer) Pm_requestGas(addr string) (bool, error) {
	account, err := (&models.Account{}).FindByAddress(s.Container.GetRepository(), strings.ToLower(addr))
	if nil != err {
		logger.S().Errorf("Query account error: %v", err)
		return false, err
	}
	var lastVip int64 = -1
	index, err := s.VipContract.TokenOfOwnerByIndex(nil, common.HexToAddress(addr), big.NewInt(0))
	if err != nil {
		// mute logs
		// logger.S().Errorf("Query account vip nft error: %v", err)
	} else {
		lastVip = index.Int64()
	}

	gas := s.MaxGas
	if lastVip != -1 {
		last, err := (&models.Account{}).FindByVipID(s.Container.GetRepository(), lastVip)
		if nil != err {
			logger.S().Errorf("Query account by vip id error: %v", err)
			return false, err
		}
		if last != nil && account.LastRequest.Unix()+86400 > time.Now().Unix() {
			return false, errors.New("frequent requests with NFT")
		}
		gas = s.MaxVipGas
	}
	if account != nil {
		if !account.Enable {
			return false, errors.New("account disabled")
		}
		if account.LastRequest.Unix()+86400 > time.Now().Unix() {
			return false, errors.New("frequent requests")
		}
	} else {
		if lastVip == -1 {
			gas = s.CreateGas
		}
		account = &models.Account{
			Address: strings.ToLower(addr),
			Enable:  true,
			UsedGas: "0",
		}
	}

	account.RemainGas = gas.String()
	account.LastRequest = time.Now()
	account.VipID = lastVip
	err = s.Container.GetRepository().Save(account).Error
	if nil != err {
		logger.S().Errorf("save account error: %v", err)
		return false, err
	}

	return true, nil
}

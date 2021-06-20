package appos

import (
	"errors"
	"runtime"

	"github.com/danieljoos/wincred"
)

type KeyChainManager interface {
	SetToken(token string) error
	UpdateToken(token string) (string, string, error)
	GetToken() (string, error)
	GetTokenAndDesc() (string, string, error)
	DeleteToken() error
}

type WindowsKeyChainManager struct {
	targetName string
}

func NewKeyChainManager() (KeyChainManager, error) {
	runtimeOs := runtime.GOOS

	switch runtimeOs {
	case "windows":
		return &WindowsKeyChainManager{
			targetName: "com.ajizablg.ojm-drone/access-token",
		}, nil

	case "darwin":
		fallthrough
	case "linux":
		fallthrough
	default:
		return nil, errors.New("your OS is not supported")
	}
}

func (km *WindowsKeyChainManager) SetToken(token string) error {

	cred := wincred.NewGenericCredential(km.targetName)
	cred.CredentialBlob = []byte(token)
	err := cred.Write()

	if err != nil {
		return err
	}

	return nil
}

func (km *WindowsKeyChainManager) UpdateToken(token string) (string, string, error) {

	desc := makeTokenDesc(token)
	err := km.SetToken(token)

	if err != nil {
		return token, desc, err
	}

	return token, desc, err
}

func (km *WindowsKeyChainManager) GetToken() (string, error) {

	cred, err := wincred.GetGenericCredential(km.targetName)
	if err != nil {
		return "", nil
	}

	return string(cred.CredentialBlob), nil
}

func (km *WindowsKeyChainManager) GetTokenAndDesc() (string, string, error) {

	token, err := km.GetToken()
	if err != nil {
		return "", "", err
	}

	return token, makeTokenDesc(token), nil
}

func (km *WindowsKeyChainManager) DeleteToken() error {

	cred, err := wincred.GetGenericCredential(km.targetName)
	if err != nil {
		return err

	}

	err = cred.Delete()
	if err != nil {
		return err
	}

	return nil
}

func makeTokenDesc(token string) string {
	desc := ""
	if len(token) > 0 {
		desc = "**********..."
		if len(token) > 10 {
			desc = token[0:5] + desc
		}
	}
	return desc
}

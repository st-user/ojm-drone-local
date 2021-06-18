package appos

import (
	"errors"
	"os"
	"runtime"

	"github.com/danieljoos/wincred"
	"github.com/keybase/go-keychain"
)

type KeyChainManager interface {
	SetToken(token string) error
	UpdateToken(token string) (string, string, error)
	GetToken() (string, error)
	GetTokenAndDesc() (string, string, error)
	DeleteToken() error
}

type MacOSKeyChainManager struct {
	service string
	group   string
	account string
	label   string
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
		return &MacOSKeyChainManager{
			service: "com.ajizablg.ojm-drone/access-token",
			group:   "com.ajizablg.ojm-drone",
			account: os.Getenv("USER"),
			label:   "OJM-Drone Access Token",
		}, nil
	case "linux":
		fallthrough
	default:
		return nil, errors.New("your OS is not supported")
	}
}

func (km *MacOSKeyChainManager) SetToken(token string) error {

	item := keychain.NewGenericPassword(
		km.service, km.account, km.label, []byte(token), km.group)
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleAfterFirstUnlockThisDeviceOnly)
	err := keychain.AddItem(item)

	if err != nil {
		return err
	}

	return nil
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

func (km *MacOSKeyChainManager) UpdateToken(token string) (string, string, error) {

	desc := makeTokenDesc(token)

	existingToken, err := km.GetToken()
	if err != nil {
		return token, desc, err
	}

	if len(existingToken) > 0 {
		err = km.DeleteToken()
		if err != nil {
			return token, desc, err
		}
	}

	err = km.SetToken(token)

	if err != nil {
		return token, desc, err
	}

	return token, desc, err
}

func (km *WindowsKeyChainManager) UpdateToken(token string) (string, string, error) {

	desc := makeTokenDesc(token)
	err := km.SetToken(token)

	if err != nil {
		return token, desc, err
	}

	return token, desc, err
}

func (km *MacOSKeyChainManager) GetToken() (string, error) {

	token, err := keychain.GetGenericPassword(km.service, km.account, km.label, km.group)
	if err != nil {
		return "", err
	}

	return string(token), nil
}

func (km *WindowsKeyChainManager) GetToken() (string, error) {

	cred, err := wincred.GetGenericCredential(km.targetName)
	if err != nil {
		return "", nil
	}

	return string(cred.CredentialBlob), nil
}

func (km *MacOSKeyChainManager) GetTokenAndDesc() (string, string, error) {

	token, err := km.GetToken()
	if err != nil {
		return "", "", err
	}

	return token, makeTokenDesc(token), nil
}

func (km *WindowsKeyChainManager) GetTokenAndDesc() (string, string, error) {

	token, err := km.GetToken()
	if err != nil {
		return "", "", err
	}

	return token, makeTokenDesc(token), nil
}

func (km *MacOSKeyChainManager) DeleteToken() error {

	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(km.service)
	item.SetAccount(km.account)
	item.SetAccessGroup(km.group)
	item.SetLabel(km.label)
	err := keychain.DeleteItem(item)

	if err != nil {
		return err
	}

	return nil
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

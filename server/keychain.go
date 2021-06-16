package main

import (
	"os"

	"github.com/keybase/go-keychain"
)

type KeyChainManager struct {
	service string
	group   string
	account string
	label   string
}

func NewKeyChainManager() KeyChainManager {
	return KeyChainManager{
		service: "com.ajizablg.ojm-drone/access-token",
		group:   "com.ajizablg.ojm-drone",
		account: os.Getenv("USER"),
		label:   "OJM-Drone Access Token",
	}
}

func (km *KeyChainManager) SetToken(token string) error {

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

func (km *KeyChainManager) UpdateToken(token string) (string, string, error) {

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

func (km *KeyChainManager) GetToken() (string, error) {

	token, err := keychain.GetGenericPassword(km.service, km.account, km.label, km.group)
	if err != nil {
		return "", err
	}

	return string(token), nil
}

func (km *KeyChainManager) GetTokenAndDesc() (string, string, error) {

	token, err := km.GetToken()
	if err != nil {
		return "", "", err
	}

	return token, makeTokenDesc(token), nil
}

func (km *KeyChainManager) DeleteToken() error {

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

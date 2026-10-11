package model

import (
	"strings"

	"gorm.io/gorm"
)

const channelEncryptedPrefix = "enc:v1:"

// EncryptChannelKey encrypts a plaintext channel key using AES-256-GCM.
func EncryptChannelKey(plaintextKey string) (string, error) {
	return EncryptTokenKey(plaintextKey)
}

// DecryptChannelKey decrypts an encrypted channel key. If the key does not start with the encrypted prefix,
// it is returned as-is to preserve backward compatibility with legacy unencrypted keys.
func DecryptChannelKey(encryptedKey string) (string, error) {
	return DecryptTokenKey(encryptedKey)
}

func (channel *Channel) BeforeCreate(tx *gorm.DB) error {
	if channel.Key == "" {
		return nil
	}
	if !strings.HasPrefix(channel.Key, channelEncryptedPrefix) {
		encrypted, err := EncryptChannelKey(channel.Key)
		if err != nil {
			return err
		}
		channel.Key = encrypted
	}
	return nil
}

func (channel *Channel) AfterCreate(tx *gorm.DB) error {
	if strings.HasPrefix(channel.Key, channelEncryptedPrefix) {
		decrypted, err := DecryptChannelKey(channel.Key)
		if err == nil {
			channel.Key = decrypted
		}
	}
	return nil
}

func (channel *Channel) BeforeUpdate(tx *gorm.DB) error {
	if channel.Key != "" && !strings.HasPrefix(channel.Key, channelEncryptedPrefix) {
		encrypted, err := EncryptChannelKey(channel.Key)
		if err != nil {
			return err
		}
		channel.Key = encrypted
	}
	return nil
}

func (channel *Channel) AfterUpdate(tx *gorm.DB) error {
	if strings.HasPrefix(channel.Key, channelEncryptedPrefix) {
		decrypted, err := DecryptChannelKey(channel.Key)
		if err == nil {
			channel.Key = decrypted
		}
	}
	return nil
}

func (channel *Channel) AfterFind(tx *gorm.DB) error {
	if strings.HasPrefix(channel.Key, channelEncryptedPrefix) {
		decrypted, err := DecryptChannelKey(channel.Key)
		if err == nil {
			channel.Key = decrypted
		}
	}
	return nil
}

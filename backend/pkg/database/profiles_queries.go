package database

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Changes interface {
	ChangeBio(newBio string) error
	ChangeNickname(newNickname string) error
	ChangeFamilyStatus(newStatus string) error
	ChangeLinks(links Links) error
}

type Links struct {
	Discord  string
	Telegram string
	Other    string
}
type UserData struct {
	Username     string
	ServerDomain string
}

func (u *User) ChangeBio(newBio string) error {
	// проверка длины
	if len(newBio) > 700 || len(newBio) == 0 {
		return fmt.Errorf("bio must be less than 700 characters")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	query := "UPDATE profiles SET bio=$1 WHERE username=$2(SELECT EXIST FROM profiles WHERE username=$2)"
	affected, err := Pool.Exec(ctx, query, newBio, u.Username)
	if err != nil {
		return err
	}

	if affected.RowsAffected() == 0 {
		return fmt.Errorf("user profile dont exist")
	}
	return nil
}
func (u *User) ChangeNickcname(new string) error {
	// проверка длины
	if len(new) > 20 || len(new) == 0 {
		return fmt.Errorf("bio must be less than 700 characters")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	query := "UPDATE profiles SET nicname=$1 WHERE username=$2(SELECT EXIST FROM profiles WHERE username=$2)"
	affected, err := Pool.Exec(ctx, query, new, u.Username)
	if err != nil {
		return err
	}
	if affected.RowsAffected() == 0 {
		return fmt.Errorf("user profile dont exist")
	}
	return nil
}

func (u *User) ChangeFamilyStatus(new string) error {
	// проверка длины
	if len(new) > 20 || len(new) == 0 {
		return fmt.Errorf("bio must be less than 700 characters")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	query := "UPDATE profiles SET family_status=$1 WHERE username=$2(SELECT EXIST FROM profiles WHERE username=$2)"
	affected, err := Pool.Exec(ctx, query, new, u.Username)
	if err != nil {
		return err
	}
	if affected.RowsAffected() == 0 {
		return fmt.Errorf("user profile dont exist")
	}
	return nil
}

// validateDiscordLink проверяет, что ссылка discord валидна
func validateDiscordLink(link string) bool {
	if link == "" {
		return false // пустая ссылка разрешена
	}
	return strings.HasPrefix(link, "https://discord.gg/") || strings.HasPrefix(link, "https://discord.com/")
}

// validateTelegramLink проверяет, что ссылка telegram валидна
func validateTelegramLink(link string) bool {
	if link == "" {
		return false // пустая ссылка разрешена
	}
	return strings.HasPrefix(link, "https://t.me/")
}

func (u *User) ChangeLinks(new Links) error {
	// проверка длины
	if new.Discord == "" && new.Telegram == "" && new.Other == "" {
		return fmt.Errorf("hollow links request")
	}

	// валидация discord ссылки
	if !validateDiscordLink(new.Discord) {
		return fmt.Errorf("invalid discord link: must start with https://discord.gg/ or https://discord.com/")
	}

	// валидация telegram ссылки
	if !validateTelegramLink(new.Telegram) {
		return fmt.Errorf("invalid telegram link: must start with https://t.me/")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	query := "UPDATE profiles SET discord=$1, telegram=$2, other=$3 WHERE username=$4 AND EXISTS(SELECT 1 FROM profiles WHERE username=$4)"
	affected, err := Pool.Exec(ctx, query, new.Discord, new.Telegram, new.Other, u.Username)
	if err != nil {
		return err
	}
	if affected.RowsAffected() == 0 {
		return fmt.Errorf("user profile dont exist")
	}
	return nil
}

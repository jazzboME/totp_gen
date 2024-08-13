package main

import (
	"log"

	"encoding/json"
	"github.com/zalando/go-keyring"
)

func getSites (service, user string) map[string]string {
	v := make(map[string]string)
	sites, err := keyring.Get(service, user)
	if err != nil {
		log.Println("No existing sites found.")
		return v
	}

	err = json.Unmarshal([]byte(sites), &v)
	if err != nil {
		log.Println("Sites could not be parsed")
	}

	return v
}

func saveSites(service, user string, m map[string]string) error {
	jsonString, err := json.Marshal(m)
	if err != nil {
		return err
	}
	err = keyring.Set(service, user, string(jsonString))
	if err != nil {
		log.Println("Failed to save secret")
		return err
	}
	return nil
}

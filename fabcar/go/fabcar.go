/*
Copyright 2020 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

func main() {
	os.Setenv("DISCOVERY_AS_LOCALHOST", "true")
	wallet, err := gateway.NewFileSystemWallet("wallet")
	if err != nil {
		fmt.Printf("Failed to create wallet: %s\n", err)
		os.Exit(1)
	}

	if !wallet.Exists("appUser") {
		err = populateWallet(wallet)
		if err != nil {
			fmt.Printf("Failed to populate wallet contents: %s\n", err)
			os.Exit(1)
		}
	}

	ccpPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org4.example.com",
		"connection-org4.yaml",
	)

	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, "appUser"),
	)
	if err != nil {
		fmt.Printf("Failed to connect to gateway: %s\n", err)
		os.Exit(1)
	}
	defer gw.Close()

	network, err := gw.GetNetwork("mychannel")
	if err != nil {
		fmt.Printf("Failed to get network: %s\n", err)
		os.Exit(1)
	}

	contract := network.GetContract("fabcar")

login:
	for {
		var username, passowrd string
		fmt.Printf("Enter username: ")
		fmt.Scanf("%s", &username)
		fmt.Printf("Enter password: ")
		fmt.Scanf("%s", &passowrd)
		evaluateResult, _ := contract.EvaluateTransaction("Login", username, passowrd)
		result := formatJSON(evaluateResult)
		boolValue, _ := strconv.ParseBool(result)
		if boolValue == true {
			break login
		}
		fmt.Println("Wrong username or password")
	}
	var input int
loop:
	for {
		fmt.Println("Choose an option by entering a number:")
		fmt.Println("0 - Initialize ledger")
		fmt.Println("1 - Read person")
		fmt.Println("2 - Read car")
		fmt.Println("3 - Get cars by color")
		fmt.Println("4 - Get cars by color and owner")
		fmt.Println("5 - Transfer car to another owner")
		fmt.Println("6 - Add car malfunction")
		fmt.Println("7 - Repair car")
		fmt.Println("8 - Change car color")
		fmt.Println("9 - Exit")

		fmt.Scanf("%d", &input)

		switch input {
		case 0:
			fmt.Println("Initializing ledger...")
			fmt.Printf("Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")

			_, err := contract.SubmitTransaction("InitLedger")
			if err != nil {
				fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
				return
			}

			fmt.Printf("*** Transaction committed successfully\n")

		case 1:
			fmt.Printf("Enter person ID: ")
			var personID string
			fmt.Scanf("%s", &personID)
			fmt.Printf("\n--> Evaluate Transaction: ReadAsset, function returns asset attributes\n")

			evaluateResult, err := contract.EvaluateTransaction("ReadPerson", personID)
			if err != nil {
				panic(fmt.Errorf("failed to evaluate transaction: %w", err))
			}
			result := formatJSON(evaluateResult)

			fmt.Printf("*** Result:%s\n", result)

		case 2:
			fmt.Printf("Enter car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)
			fmt.Printf("\n--> Evaluate Transaction: ReadAsset, function returns asset attributes\n")

			evaluateResult, err := contract.EvaluateTransaction("ReadCar", carID)
			if err != nil {
				panic(fmt.Errorf("failed to evaluate transaction: %w", err))
			}
			result := formatJSON(evaluateResult)

			fmt.Printf("*** Result:%s\n", result)

		case 3:
			fmt.Printf("Enter car color: ")
			var color string
			fmt.Scanf("%s", &color)
			fmt.Println("Evaluate Transaction: GetCarsByColor, function returns all the cars with the given color")

			evaluateResult, err := contract.EvaluateTransaction("GetCarsByColor", color)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to evaluate transaction: %w", err))
				return
			}
			result := formatJSON(evaluateResult)

			fmt.Printf("*** Result:%s\n", result)

		case 4:
			fmt.Printf("Enter car color: ")
			var color string
			fmt.Scanf("%s", &color)

			fmt.Printf("Enter car owner: ")
			var ownerID string
			fmt.Scanf("%s", &ownerID)
			fmt.Println("Evaluate Transaction: GetCarsByColorAndOwner, function returns all the cars with the given color and owner")

			evaluateResult, err := contract.EvaluateTransaction("GetCarsByColorAndOwner", color, ownerID)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to evaluate transaction: %w", err))
				return
			}
			result := formatJSON(evaluateResult)

			fmt.Printf("*** Result:%s\n", result)

		case 5:
			fmt.Printf("Enter car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)

			fmt.Printf("Enter new owner ID: ")
			var newOwnerID string
			fmt.Scanf("%s", &newOwnerID)

			fmt.Printf("Does the owner accept malfunctioned car, with a price compensation? (Y/n): ")
			var acceptMalfunctionedStr string
			fmt.Scanf("%s", &acceptMalfunctionedStr)
			var acceptMalfunctionedBool bool
			if acceptMalfunctionedStr == "n" {
				acceptMalfunctionedBool = false
			} else {
				acceptMalfunctionedBool = true
			}

			fmt.Printf("Submit Transaction: ChangeCarOwnership, change car owner \n")

			_, err := contract.SubmitTransaction("ChangeCarOwnership", carID, newOwnerID, strconv.FormatBool(acceptMalfunctionedBool))
			if err != nil {
				fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
				return
			}

			fmt.Printf("*** Transaction committed successfully\n")

		case 6:
			fmt.Printf("Enter car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)

			fmt.Println("Enter malfunction description:")
			var description string
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				description = scanner.Text()
			}

			fmt.Printf("Enter malfunction cost: ")
			var repairPrice float64
			fmt.Scanf("%f", &repairPrice)
			fmt.Printf("Submit Transaction: AddCarMalfunction, record a new car malfunction \n")

			_, err := contract.SubmitTransaction("AddCarMalfunction", carID, description, fmt.Sprintf("%f", repairPrice))
			if err != nil {
				fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
				return
			}

			fmt.Printf("*** Transaction committed successfully\n")

		case 7:
			fmt.Printf("Enter car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)
			fmt.Printf("Submit Transaction: RepairCar, fix all of the car's malfunctions \n")

			_, err := contract.SubmitTransaction("RepairCar", carID)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
				return
			}

			fmt.Printf("*** Transaction committed successfully\n")

		case 8:
			fmt.Printf("Enter car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)

			fmt.Printf("Enter new car color: ")
			var newColor string
			fmt.Scanf("%s", &newColor)
			fmt.Printf("Submit Transaction: ChangeCarColor, change the color of a car \n")

			_, err := contract.SubmitTransaction("ChangeCarColor", carID, newColor)
			if err != nil {
				fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
				return
			}

			fmt.Printf("*** Transaction committed successfully\n")
		case 9:
			fmt.Printf("Exiting...")
			break loop

		default:
			fmt.Printf("Invalid input! Please enter a number in the range [1, 9]!")
		}

		fmt.Printf("\n\n")
	}
}

func populateWallet(wallet *gateway.Wallet) error {
	credPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org4.example.com",
		"users",
		"User1@org4.example.com",
		"msp",
	)

	certPath := filepath.Join(credPath, "signcerts", "cert.pem")
	// read the certificate pem
	cert, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	keyDir := filepath.Join(credPath, "keystore")
	// there's a single file in this dir containing the private key
	files, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return errors.New("keystore folder should have contain one file")
	}
	keyPath := filepath.Join(keyDir, files[0].Name())
	key, err := ioutil.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity("Org4MSP", string(cert), string(key))

	err = wallet.Put("appUser", identity)
	if err != nil {
		return err
	}
	return nil
}

func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}

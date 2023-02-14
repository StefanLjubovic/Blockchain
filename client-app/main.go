package main

import (
	"bufio"
	"bytes"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var now = time.Now()
var assetId = fmt.Sprintf("asset%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)

const (
	mspID        = "Org4MSP"
	cryptoPath   = "../test-network/organizations/peerOrganizations/org4.example.com"
	certPath     = cryptoPath + "/users/User1@org4.example.com/msp/signcerts/cert.pem"
	keyPath      = cryptoPath + "/users/User1@org4.example.com/msp/keystore/"
	tlsCertPath  = cryptoPath + "/peers/peer0.org4.example.com/tls/ca.crt"
	peerEndpoint = "localhost:13051"
	gatewayPeer  = "peer0.org4.example.com"
)

func main() {

	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection(tlsCertPath, gatewayPeer, peerEndpoint)
	defer clientConnection.Close()

	id := newIdentity(certPath, mspID)
	sign := newSign(keyPath)

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	chaincodeName := "basic"
	channelName := "mychannel"

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

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
			initLedger(contract)

		case 1:
			fmt.Printf("Enter person ID: ")
			var personID string
			fmt.Scanf("%s", &personID)
			readPersonByID(contract, personID)

		case 2:
			fmt.Printf("Enter car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)
			readCarByID(contract, carID)

		case 3:
			fmt.Printf("Enter car color: ")
			var color string
			fmt.Scanf("%s", &color)
			getCarsByColor(contract, color)

		case 4:
			fmt.Printf("Enter car color: ")
			var color string
			fmt.Scanf("%s", &color)

			fmt.Printf("Enter car owner: ")
			var ownerID string
			fmt.Scanf("%s", &ownerID)
			getCarsByColorAndOwner(contract, color, ownerID)

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

			changeCarOwnership(contract, carID, newOwnerID, acceptMalfunctionedBool)

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
			addCarMalfunction(contract, carID, description, repairPrice)

		case 7:
			fmt.Printf("Enter car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)
			repairCar(contract, carID)

		case 8:
			fmt.Printf("Enter car ID: ")
			var carID string
			fmt.Scanf("%s", &carID)

			fmt.Printf("Enter new car color: ")
			var newColor string
			fmt.Scanf("%s", &newColor)
			changeCarColor(contract, carID, newColor)
		case 9:
			fmt.Printf("Exiting...")
			break loop

		default:
			fmt.Printf("Invalid input! Please enter a number in the range [1, 9]!")
		}

		fmt.Printf("\n\n")
	}
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection(tlsCertPath string, gatewayPeer string, peerEndpoint string) *grpc.ClientConn {
	certificate, err := loadCertificate(tlsCertPath)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM)
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity(certPath string, mspID string) *identity.X509Identity {
	certificate, err := loadCertificate(certPath)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign(keyPath string) identity.Sign {
	files, err := os.ReadDir(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key directory: %w", err))
	}
	privateKeyPEM, err := os.ReadFile(path.Join(keyPath, files[0].Name()))

	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

// This type of transaction would typically only be run once by an application the first time it was started after its
// initial deployment. A new version of the chaincode deployed later would likely not need to run an "init" function.
func initLedger(contract *client.Contract) {
	fmt.Printf("Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Evaluate a transaction by assetID to query ledger state.
func readCarByID(contract *client.Contract, id string) {
	fmt.Printf("\n--> Evaluate Transaction: ReadAsset, function returns asset attributes\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadCar", id)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

func readPersonByID(contract *client.Contract, id string) {
	fmt.Printf("\n--> Evaluate Transaction: ReadAsset, function returns asset attributes\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadPerson", id)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

func getCarsByColor(contract *client.Contract, color string) {
	fmt.Println("Evaluate Transaction: GetCarsByColor, function returns all the cars with the given color")

	evaluateResult, err := contract.EvaluateTransaction("GetCarsByColor", color)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to evaluate transaction: %w", err))
		return
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

func getCarsByColorAndOwner(contract *client.Contract, color string, ownerId string) {
	fmt.Println("Evaluate Transaction: GetCarsByColorAndOwner, function returns all the cars with the given color and owner")

	evaluateResult, err := contract.EvaluateTransaction("GetCarsByColorAndOwner", color, ownerId)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to evaluate transaction: %w", err))
		return
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

func changeCarOwnership(contract *client.Contract, id string, newOwner string, acceptMalfunction bool) {
	fmt.Printf("Submit Transaction: ChangeCarOwnership, change car owner \n")

	_, err := contract.SubmitTransaction("ChangeCarOwnership", id, newOwner, strconv.FormatBool(acceptMalfunction))
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func addCarMalfunction(contract *client.Contract, id string, description string, repairPrice float64) {
	fmt.Printf("Submit Transaction: AddCarMalfunction, record a new car malfunction \n")

	_, err := contract.SubmitTransaction("AddCarMalfunction", id, description, fmt.Sprintf("%f", repairPrice))
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func repairCar(contract *client.Contract, id string) {
	fmt.Printf("Submit Transaction: RepairCar, fix all of the car's malfunctions \n")

	_, err := contract.SubmitTransaction("RepairCar", id)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func changeCarColor(contract *client.Contract, id string, newColor string) {
	fmt.Printf("Submit Transaction: ChangeCarColor, change the color of a car \n")

	_, err := contract.SubmitTransaction("ChangeCarColor", id, newColor)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to submit transaction: %w", err))
		return
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}

package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SimpleChaincode struct {
	contractapi.Contract
}

const compositeIndex = "color-ownerId-Id"

func (s *SimpleChaincode) InitLedger(ctx contractapi.TransactionContextInterface) error {

	cars := []Car{
		{Id: "1", Price: 6400.00, Brand: "Fiat", Model: "punto", Year: 2014, Color: "white", OwnerId: "1", Malfuncations: []Malfunction{
			{Description: "Some malfunction", Cost: 3000},
			{Description: "Malfunction two", Cost: 2000},
		}},
		{Id: "2", Price: 7500.00, Brand: "Ford", Model: "fiesta", Year: 2015, Color: "blue", OwnerId: "2", Malfuncations: []Malfunction{
			{Description: "Next malfunction", Cost: 4000},
			{Description: "Malfunction third", Cost: 5000},
		}},
		{Id: "3", Price: 6400.00, Brand: "Opel", Model: "Astra", Year: 2015, Color: "blue", OwnerId: "2", Malfuncations: []Malfunction{
			{Description: "Next malfunction", Cost: 4000},
		}},
		{Id: "4", Price: 4000.00, Brand: "Renault", Model: "Clio", Year: 2015, Color: "blue", OwnerId: "3", Malfuncations: []Malfunction{}},
		{Id: "5", Price: 5500.00, Brand: "BMW", Model: "i3", Year: 2015, Color: "blue", OwnerId: "3", Malfuncations: []Malfunction{
			{Description: "Malfunction third", Cost: 5000},
		}},
		{Id: "6", Brand: "Skoda", Model: "Fabia", Year: 2015, Color: "blue", OwnerId: "1", Malfuncations: []Malfunction{}, Price: 7000.00},
	}
	persons := []Person{
		{Id: "1", Name: "Mika", Surname: "Mikic", Address: "mika@gmal.com", MoneySum: 6700.00},
		{Id: "2", Name: "Pera", Surname: "Peric", Address: "pera@gmal.com", MoneySum: 7700.00},
		{Id: "3", Name: "Ivo", Surname: "Ban", Address: "ivo@gmal.com", MoneySum: 4400.70},
	}

	for _, personAsset := range persons {
		personJSON, err := json.Marshal(personAsset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(personAsset.Id, personJSON)
		if err != nil {
			return fmt.Errorf("failed to put person to world state. %v", err)
		}
	}

	for _, car := range cars {
		carJSON, err := json.Marshal(car)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(car.Id, carJSON)
		if err != nil {
			return fmt.Errorf("failed to put cars to world state. %v", err)
		}

		key, err := ctx.GetStub().CreateCompositeKey(compositeIndex, []string{car.Color, car.OwnerId, car.Id})
		if err != nil {
			return err
		}
		value := []byte{0x00}
		err = ctx.GetStub().PutState(key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SimpleChaincode) ReadPerson(ctx contractapi.TransactionContextInterface, id string) (*Person, error) {
	personJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read person from world state: %v", err)
	}
	if personJSON == nil {
		return nil, fmt.Errorf("the person %s does not exist", id)
	}

	var person Person
	err = json.Unmarshal(personJSON, &person)
	if err != nil {
		return nil, err
	}

	return &person, nil
}

func (s *SimpleChaincode) ReadCar(ctx contractapi.TransactionContextInterface, id string) (*Car, error) {
	carJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read car from world state: %v", err)
	}
	if carJSON == nil {
		return nil, fmt.Errorf("the car %s does not exist", id)
	}

	var car Car
	err = json.Unmarshal(carJSON, &car)
	if err != nil {
		return nil, err
	}

	return &car, nil
}

func (s *SimpleChaincode) GetCarsByColor(ctx contractapi.TransactionContextInterface, color string) ([]*Car, error) {
	coloredCarIter, err := ctx.GetStub().GetStateByPartialCompositeKey(compositeIndex, []string{color})
	if err != nil {
		return nil, err
	}

	defer coloredCarIter.Close()

	var list = []*Car{}

	for i := 0; coloredCarIter.HasNext(); i++ {
		responseRange, err := coloredCarIter.Next()
		if err != nil {
			return nil, err
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return nil, err
		}

		retCarID := compositeKeyParts[2]

		car, err := s.ReadCar(ctx, retCarID)
		if err != nil {
			return nil, err
		}

		list = append(list, car)
	}

	return list, nil
}

func (s *SimpleChaincode) GetCarsByColorAndOwner(ctx contractapi.TransactionContextInterface, color string, ownerID string) ([]*Car, error) {
	_, err := s.ReadPerson(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	coloredCarByOwnerIter, err := ctx.GetStub().GetStateByPartialCompositeKey(compositeIndex, []string{color, ownerID})
	if err != nil {
		return nil, err
	}

	defer coloredCarByOwnerIter.Close()

	var list = []*Car{}

	for i := 0; coloredCarByOwnerIter.HasNext(); i++ {
		responseRange, err := coloredCarByOwnerIter.Next()
		if err != nil {
			return nil, err
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return nil, err
		}

		retCarID := compositeKeyParts[2]

		carAsset, err := s.ReadCar(ctx, retCarID)
		if err != nil {
			return nil, err
		}

		list = append(list, carAsset)
	}

	return list, nil
}

func (s *SimpleChaincode) ChangeCarOwnership(ctx contractapi.TransactionContextInterface, carId string, newOwnerId string, acceptMalfunction bool) (bool, error) {
	car, err := s.ReadCar(ctx, carId)
	if err != nil {
		return false, err
	}

	if !acceptMalfunction {
		return false, fmt.Errorf("Malfunctions are not accepted")
	}

	if car.OwnerId == newOwnerId {
		return false, fmt.Errorf("Person is already owner of the car!")
	}

	buyer, err := s.ReadPerson(ctx, newOwnerId)
	if err != nil {
		return false, err
	}

	seller, err := s.ReadPerson(ctx, car.OwnerId)
	if err != nil {
		return false, err
	}

	price := 0.00

	if len(car.Malfuncations) == 0 {
		price = car.Price
	} else {
		for _, carMalfunction := range car.Malfuncations {
			price -= carMalfunction.Cost
		}
	}
	if buyer.MoneySum >= price {
		buyer.MoneySum -= price
		seller.MoneySum += price
	} else {
		return false, fmt.Errorf("the buyer does not own enough money to purchase the car")
	}

	oldOwnerID := car.OwnerId
	car.OwnerId = newOwnerId

	carJSON, err := json.Marshal(car)
	if err != nil {
		return false, err
	}

	buyerJSON, err := json.Marshal(buyer)
	if err != nil {
		return false, err
	}

	sellerJSON, err := json.Marshal(seller)
	if err != nil {
		return false, err
	}

	err = ctx.GetStub().PutState(carId, carJSON)
	if err != nil {
		return false, err
	}

	err = ctx.GetStub().PutState(buyer.Id, buyerJSON)
	if err != nil {
		return false, err
	}

	err = ctx.GetStub().PutState(seller.Id, sellerJSON)
	if err != nil {
		return false, err
	}

	colorNewOwnerIndexKey, err := ctx.GetStub().CreateCompositeKey(compositeIndex, []string{car.Color, newOwnerId, car.Id})
	if err != nil {
		return false, err
	}

	value := []byte{0x00}
	err = ctx.GetStub().PutState(colorNewOwnerIndexKey, value)
	if err != nil {
		return false, err
	}

	colorOldOwnerIndexKey, err := ctx.GetStub().CreateCompositeKey(compositeIndex, []string{car.Color, oldOwnerID, car.Id})
	if err != nil {
		return false, err
	}

	err = ctx.GetStub().DelState(colorOldOwnerIndexKey)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *SimpleChaincode) AddCarMalfunction(ctx contractapi.TransactionContextInterface, id string, description string, cost float64) error {
	car, err := s.ReadCar(ctx, id)
	if err != nil {
		return err
	}

	newMalfunction := Malfunction{
		Description: description,
		Cost:        cost,
	}

	car.Malfuncations = append(car.Malfuncations, newMalfunction)

	totalRepairPrice := 0.00
	for _, carMalfunction := range car.Malfuncations {
		totalRepairPrice += carMalfunction.Cost
	}

	if totalRepairPrice > car.Price {
		return ctx.GetStub().DelState(id)
	}

	carJSON, err := json.Marshal(car)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, carJSON)
	if err != nil {
		return err
	}

	return nil
}

func (s *SimpleChaincode) RepairCar(ctx contractapi.TransactionContextInterface, id string) error {
	car, err := s.ReadCar(ctx, id)
	if err != nil {
		return err
	}

	person, err := s.ReadPerson(ctx, car.OwnerId)
	if err != nil {
		return err
	}

	repairPrice := 0.00
	for _, malfunction := range car.Malfuncations {
		repairPrice += malfunction.Cost
	}
	if repairPrice > person.MoneySum {
		return fmt.Errorf("Not enough money for repair")
	}

	car.Malfuncations = []Malfunction{}
	person.MoneySum -= repairPrice

	carJSON, err := json.Marshal(car)
	if err != nil {
		return err
	}

	personJSON, err := json.Marshal(person)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, carJSON)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(person.Id, personJSON)
	if err != nil {
		return err
	}

	return nil
}

func (s *SimpleChaincode) ChangeCarColor(ctx contractapi.TransactionContextInterface, id string, newColor string) (string, error) {
	car, err := s.ReadCar(ctx, id)
	if err != nil {
		return "", err
	}

	oldColor := car.Color
	car.Color = newColor

	carJSON, err := json.Marshal(car)
	if err != nil {
		return "", err
	}

	err = ctx.GetStub().PutState(id, carJSON)
	if err != nil {
		return "", err
	}

	newColorOwnerIndexKey, err := ctx.GetStub().CreateCompositeKey(compositeIndex, []string{newColor, car.OwnerId, car.Id})
	if err != nil {
		return "", err
	}

	value := []byte{0x00}
	err = ctx.GetStub().PutState(newColorOwnerIndexKey, value)
	if err != nil {
		return "", err
	}

	oldColorOwnerIndexKey, err := ctx.GetStub().CreateCompositeKey(compositeIndex, []string{oldColor, car.OwnerId, car.Id})
	if err != nil {
		return "", err
	}

	err = ctx.GetStub().DelState(oldColorOwnerIndexKey)
	if err != nil {
		return "", err
	}

	return oldColor, nil
}

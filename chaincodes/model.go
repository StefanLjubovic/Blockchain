package main

type Person struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Surname  string  `json:"surname"`
	Address  string  `json:"address"`
	MoneySum float64 `json:"moneySum"`
}

type Car struct {
	Id            string        `json:"id"`
	Brand         string        `json:"brand"`
	Model         string        `json:"model"`
	Year          int           `json:"year"`
	Color         string        `json:"color"`
	OwnerId       string        `json:"ownerId"`
	Malfuncations []Malfunction `json:"malfunctions"`
	Price         float64       `json:"price"`
}

type Malfunction struct {
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
}

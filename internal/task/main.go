package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Event struct {
	ID       uint    `json:"id"`
	CarModel string  `json:"car_model"`
	Price    float64 `json:"price"`
}

type producer struct {
	file    *os.File
	encoder *json.Encoder
}

func NewProducer(filename string) (*producer, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	return &producer{file: file, encoder: json.NewEncoder(file)}, nil
	// допишите код здесь
}

func (p *producer) WriteEvent(event *Event) error {
	return p.encoder.Encode(event)
	// допишите код здесь
}

func (p *producer) Close() error {
	return p.file.Close()
}

type consumer struct {
	file    *os.File
	decoder *json.Decoder
}

func NewConsumer(filename string) (*consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}
	return &consumer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
	// допишите код здесь
}

func (c *consumer) ReadEvent() (*Event, error) {
	event := &Event{}
	err := c.decoder.Decode(&event)
	if err != nil {
		return nil, err
	}
	return event, nil
	// допишите код здесь
}

func (c *consumer) Close() error {
	return c.file.Close()
}

var events = []*Event{
	{
		ID:       1,
		CarModel: "Lada",
		Price:    400000,
	},
	{
		ID:       2,
		CarModel: "Mitsubishi",
		Price:    650000,
	},
	{
		ID:       3,
		CarModel: "Toyota",
		Price:    800000,
	},
	{
		ID:       4,
		CarModel: "BMW",
		Price:    875000,
	},
	{
		ID:       5,
		CarModel: "Mercedes",
		Price:    999999,
	},
}

func main() {
	filename := "colz_events.log"
	defer os.Remove(filename)

	producer, err := NewProducer(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	consumer, err := NewConsumer(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	for _, event := range events {
		err := producer.WriteEvent(event)
		if err != nil {
			log.Fatal(err)
		}

		readedEvent, err := consumer.ReadEvent()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(readedEvent)
	}
}

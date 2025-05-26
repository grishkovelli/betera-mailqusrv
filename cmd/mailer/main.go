package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-faker/faker/v4"

	"mailqusrv/internal/entities"
)

func main() {
	requests := 50

	wg := sync.WaitGroup{}
	wg.Add(requests)
	for range requests {
		go func() {
			body := entities.CreateEmail{
				To:      randomEmail(),
				Subject: faker.Word(),
				Body:    faker.Word(),
			}

			jsn, err := json.Marshal(body)
			if err != nil {
				log.Fatalf("Failed serialization: %v", err)
			}

			resp, err := http.Post("http://app:3000/send-email", "application/json", bytes.NewBuffer(jsn))
			if err != nil {
				log.Fatalf("Failed request %v", err)
			}
			defer resp.Body.Close()
			defer wg.Done()
		}()
	}

	wg.Wait()
}

func randomEmail() string {
	return strings.ToLower(
		fmt.Sprintf("%s-%s@%s", faker.FirstName(), faker.LastName(), faker.DomainName()),
	)
}

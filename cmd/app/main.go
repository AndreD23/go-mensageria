package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AndreD23/go-mensageria/internal/infra/akafka"
	"github.com/AndreD23/go-mensageria/internal/infra/repository"
	"github.com/AndreD23/go-mensageria/internal/infra/web"
	"github.com/AndreD23/go-mensageria/internal/usecase"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-chi/chi/v5"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("Hello there")

	db, err := sql.Open("mysql", "root:root@tcp(host.docker.internal:3306)/products")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	repository := repository.NewProductRepositoryMysql(db)
	createProductUseCase := usecase.NewCreateProductUseCase(repository)
	listProductUseCase := usecase.NewListProductsUsecase(repository)

	productHandlers := web.NewProductHandlers(createProductUseCase, listProductUseCase)

	r := chi.NewRouter()
	r.Post("/products", productHandlers.CreateProductUseCase)
	r.Get("/products", productHandlers.ListProductsUsecase)

	go http.ListenAndServe(":8000", r)

	msgChan := make(chan *kafka.Message)
	// Joga em outra thread para não travar a aplicação, pois isto ficará em looping infinito
	go akafka.Consume([]string{"products"}, "host.docker.internal:9094", msgChan)

	for msg := range msgChan {
		dto := usecase.CreateProductInputDTO{}
		err := json.Unmarshal(msg.Value, &dto)
		if err != nil {
			panic(err)
		}
		_, err = createProductUseCase.Execute(dto)
		if err != nil {
			panic(err)
		}
	}

}

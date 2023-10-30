package main

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Produto struct {
	gorm.Model
	ID          uint
	NomeProduto string
	ValorCompra float64
	ValorVenda  float64
}

type ProdutoPageData struct {
	PageTitle string
	Produtos  []Produto
}

func main() {
	db, err := gorm.Open(sqlite.Open("product.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect database")
	}
	db.AutoMigrate(&Produto{})

	var count int64
	db.Model(&Produto{}).Count(&count)

	if count == 0 {
		// Se a tabela estiver vazia, insira alguns produtos iniciais
		initialProdutos := []Produto{
			{NomeProduto: "Café Espresso", ValorCompra: 2.50, ValorVenda: 3.50},
			{NomeProduto: "Cappuccino", ValorCompra: 3.00, ValorVenda: 4.50},
			{NomeProduto: "Café Americano", ValorCompra: 2.00, ValorVenda: 3.00},
			{NomeProduto: "Latte Macchiato", ValorCompra: 3.50, ValorVenda: 4.50},
			{NomeProduto: "Mocha", ValorCompra: 3.50, ValorVenda: 4.75},
			{NomeProduto: "Chá Verde", ValorCompra: 2.00, ValorVenda: 3.25},
			{NomeProduto: "Chá de Camomila", ValorCompra: 2.00, ValorVenda: 3.25},
			{NomeProduto: "Café Descafeinado", ValorCompra: 2.75, ValorVenda: 4.00},
			{NomeProduto: "Café com Leite", ValorCompra: 3.00, ValorVenda: 4.25},
			{NomeProduto: "Chocolatte", ValorCompra: 3.50, ValorVenda: 4.75},
			{NomeProduto: "Croissant", ValorCompra: 2.50, ValorVenda: 3.75},
			{NomeProduto: "Muffin de Blueberry", ValorCompra: 2.75, ValorVenda: 4.00},
			{NomeProduto: "Sanduíche de Presunto e Queijo", ValorCompra: 4.50, ValorVenda: 6.00},
			{NomeProduto: "Bolo de Chocolate", ValorCompra: 3.25, ValorVenda: 4.75},
			{NomeProduto: "Bolo de Cenoura", ValorCompra: 3.25, ValorVenda: 4.75},
			{NomeProduto: "Torta de Maçã", ValorCompra: 3.50, ValorVenda: 5.25},
			{NomeProduto: "Donuts", ValorCompra: 1.50, ValorVenda: 2.75},
			{NomeProduto: "Água Mineral", ValorCompra: 1.00, ValorVenda: 1.75},
			{NomeProduto: "Suco de Laranja", ValorCompra: 2.50, ValorVenda: 4.00},
			{NomeProduto: "Smoothie de Frutas", ValorCompra: 3.50, ValorVenda: 5.25},
		}

		for _, produto := range initialProdutos {
			db.Create(&produto)
		}
	}

	r := mux.NewRouter()
	r.HandleFunc("/", ListProdutosHandler).Methods("GET")

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

func ListProdutosHandler(w http.ResponseWriter, r *http.Request) {
	db, err := gorm.Open(sqlite.Open("product.db"), &gorm.Config{})
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}

	var produtos []Produto
	db.Find(&produtos)

	tmpl := template.Must(template.ParseFiles("template/index.html"))
	data := ProdutoPageData{
		PageTitle: "Coffee Shop - Manutenção de Estoque",
		Produtos:  produtos,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

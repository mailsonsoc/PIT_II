package main

import (
	"fmt"
	"html/template"
	"net/http"

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
	Produto   Produto
}

func main() {

	// Configuração do servidor de arquivos estáticos
	fs := http.FileServer(http.Dir("template"))
	http.Handle("/", fs)

	// Roteamento para diferentes endpoints
	http.HandleFunc("/pagina_inicial", paginaInicialHandler)
	http.HandleFunc("/catalogo", catalogoHandler)
	http.HandleFunc("/sobre_nos", sobreNosHandler)
	http.HandleFunc("/fale_conosco", faleConoscoHandler)
	http.HandleFunc("/carrinho", carrinhoHandler)

	// Definindo o endereço e porta do servidor
	port := ":8081"

	// Iniciando o servidor
	conn := http.ListenAndServe(port, nil)
	if conn != nil {
		panic(conn)
	}
}

// Handlers para cada endpoint
func paginaInicialHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "template/index.html")
}

func catalogoHandler(w http.ResponseWriter, r *http.Request) {
	// Abre o banco de dados no diretório atual
	db, err := gorm.Open(sqlite.Open("C:/Users/albuq/go/src/PIT_II/Server_Usuario/product.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect database")
	}

	// Migrate (cria) a tabela Produto se ela ainda não existir
	if err := db.AutoMigrate(&Produto{}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to migrate database: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Realiza uma busca por todos os produtos na tabela
	var produtos []Produto
	if err := db.Find(&produtos).Error; err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve data from database: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Carrega os dados na página HTML
	tmpl := template.Must(template.ParseFiles("template/catalogo.html"))
	data := ProdutoPageData{
		PageTitle: "Coffee Shop - Catalogo",
		Produtos:  produtos,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute template: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func sobreNosHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "template/sobre_nos.html")
}

func faleConoscoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "template/fale_conosco.html")
}

func carrinhoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "template/carrinho.html")
}

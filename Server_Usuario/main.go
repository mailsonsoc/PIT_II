package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// Rota para a página inicial
	r.HandleFunc("/", HomeHandler)

	// Rota para a página inicial
	r.HandleFunc("/pagina_inicial", HomeHandler)

	// Rota para a página catalogo.html
	r.HandleFunc("/catalogo", CatalogoHandler)

	// Rota para a página sobre_nos.html
	r.HandleFunc("/sobre_nos", SobreNosHandler)

	// Rota para a página fale_conosco.html
	r.HandleFunc("/fale_conosco", FaleConoscoHandler)

	// Rota para a página carrinho.html
	r.HandleFunc("/carrinho", CarrinhoHandler)

	// Manipulador para servir os arquivos estáticos do diretório /template/
	r.PathPrefix("/template/").Handler(http.StripPrefix("/template/", http.FileServer(http.Dir("./template/"))))

	// Configuração do servidor para escutar na porta 8081
	port := ":8081"
	fmt.Printf("Servidor rodando em http://localhost%s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		fmt.Println(err)
	}
}

// Handler para a página inicial
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "./template/", http.StatusSeeOther)
}

// Handler para a página catalogo.html
func CatalogoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./template/catalogo.html")
}

// Handler para a página sobre_nos.html
func SobreNosHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./template/sobre_nos.html")
}

// Handler para a página fale_conosco.html
func FaleConoscoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./template/fale_conosco.html")
}

// Handler para a página carrinho.html
func CarrinhoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./template/carrinho.html")
}

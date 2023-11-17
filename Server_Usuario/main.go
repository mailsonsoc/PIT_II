package main

import (
	"net/http"
)

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
	err := http.ListenAndServe(port, nil)
	if err != nil {
		panic(err)
	}
}

// Handlers para cada endpoint
func paginaInicialHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "template/index.html")
}

func catalogoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "template/catalogo.html")
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

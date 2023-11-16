package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Manipulador para a página inicial desejada (/template/)
	http.Handle("/", http.RedirectHandler("/template/", http.StatusSeeOther))

	// Manipulador para servir os arquivos estáticos do diretório /template/
	http.Handle("/template/", http.StripPrefix("/template/", http.FileServer(http.Dir("./template/"))))

	// Configuração do servidor para escutar na porta 8081
	port := ":8081"
	fmt.Printf("Servidor rodando em http://localhost%s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Println(err)
	}
}

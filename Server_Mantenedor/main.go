package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type FirestoreClient struct {
	Client *firestore.Client
	Ctx    context.Context
}

type Produto struct {
	ID          int
	NomeProduto string
	ValorCompra float64
	ValorVenda  float64
}

type Ticket struct {
	Titulo       string
	Descricao    string
	DataAbertura time.Time
}

type Transacao struct {
	CodigoTransacao int
	CodigoProd      int
	NomeProd        string
	QuantidadeProd  int
	ValorTransacao  float64
	DataTransacao   time.Time
}

type RelatorioPageData struct {
	Meses []string
	Anos  []string
}

type ProdutoPageData struct {
	PageTitle  string
	Produtos   []Produto
	Tickets    []Ticket
	Transacoes []Transacao
	Produto    Produto
}

type TransacaoPageData struct {
	Transacoes []Transacao
}

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", LoginHandler).Methods("GET")
	r.HandleFunc("/index", ListProdutosHandler).Methods("GET")
	r.HandleFunc("/produto/novo", CreateProdutoHandler).Methods("GET", "POST")
	r.HandleFunc("/produto/editar/{id:[0-9]+}", EditProdutoHandler).Methods("GET", "POST")
	r.HandleFunc("/produto/excluir/{id:[0-9]+}", DeleteProdutoHandler).Methods("POST")
	r.HandleFunc("/abrir-ticket", AbrirTicketHandler).Methods("GET", "POST")
	r.HandleFunc("/tickets", ListTicketsHandler).Methods("GET")
	r.HandleFunc("/relatorio-fluxo", RelatorioFluxoHandler).Methods("GET")
	r.HandleFunc("/visualizar-transacoes", VisualizarTransacoesHandler).Methods("GET")
	r.HandleFunc("/gerar-relatorio", GerarRelatorioHandler).Methods("POST") // Adicionando a rota para lidar com a submissão do formulário

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

func InitializeFirestore() (*FirestoreClient, error) {
	ctx := context.Background()

	// Substitua o caminho do seu arquivo de credenciais JSON do Firebase
	opt := option.WithCredentialsFile("C:/Users/albuq/go/fir-db-pitii-firebase-adminsdk-sok9l-acdd50458a.json")

	// Inicialize o cliente Firestore
	client, err := firestore.NewClient(ctx, "fir-db-pitii", opt)
	if err != nil {
		log.Fatalf("Erro ao inicializar o cliente Firestore: %v", err)
		return nil, err
	}

	return &FirestoreClient{
		Client: client,
		Ctx:    ctx,
	}, nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if authenticate(w, r) {
		http.Redirect(w, r, "/index", http.StatusSeeOther)
		return
	}

	// Se as credenciais estiverem incorretas, exibir uma mensagem de erro
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	http.Error(w, "Credenciais incorretas", http.StatusUnauthorized)
}

func CreateProdutoHandler(w http.ResponseWriter, r *http.Request) {
	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		nomeProduto := r.FormValue("nomeProduto")
		valorCompra := r.FormValue("valorCompra")
		valorVenda := r.FormValue("valorVenda")

		valorCompraFloat, err := strconv.ParseFloat(valorCompra, 64)
		if err != nil {
			http.Error(w, "Invalid valorCompra", http.StatusBadRequest)
			return
		}

		valorVendaFloat, err := strconv.ParseFloat(valorVenda, 64)
		if err != nil {
			http.Error(w, "Invalid valorVenda", http.StatusBadRequest)
			return
		}

		// Buscar os IDs existentes na coleção de produtos
		produtosRef := firestoreClient.Client.Collection("produtos")
		query := produtosRef.Select("ID")
		iter := query.Documents(firestoreClient.Ctx)

		var existingIDs []int
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Failed to fetch product IDs from Firestore", http.StatusInternalServerError)
				return
			}
			var produto Produto
			if err := doc.DataTo(&produto); err != nil {
				http.Error(w, "Failed to parse product data", http.StatusInternalServerError)
				return
			}
			existingIDs = append(existingIDs, int(produto.ID))
		}

		// Encontrar o próximo ID disponível para o novo produto
		newID := findAvailableID(existingIDs)

		// Criar o novo produto com o ID gerado automaticamente
		produto := Produto{
			ID:          int(newID),
			NomeProduto: nomeProduto,
			ValorCompra: valorCompraFloat,
			ValorVenda:  valorVendaFloat,
		}

		_, err = produtosRef.Doc(strconv.Itoa(newID)).Set(firestoreClient.Ctx, produto)
		if err != nil {
			log.Printf("Failed to create product in Firestore: %v", err)
			http.Error(w, "Failed to create product in Firestore", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)

	}
	tmpl := template.Must(template.ParseFiles("template/create.html"))
	data := ProdutoPageData{
		PageTitle: "Coffee Shop - Novo Produto",
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

// Função auxiliar para encontrar o próximo ID disponível
func findAvailableID(existingIDs []int) int {
	// Lógica para encontrar o próximo ID disponível, por exemplo:
	maxID := 0
	for _, id := range existingIDs {
		if id > maxID {
			maxID = id
		}
	}
	return maxID + 1
}

func EditProdutoHandler(w http.ResponseWriter, r *http.Request) {
	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		vars := mux.Vars(r)
		id := vars["id"]

		nomeProduto := r.FormValue("nomeProduto")
		valorCompra := r.FormValue("valorCompra")
		valorVenda := r.FormValue("valorVenda")
		valorCompraFloat, err := strconv.ParseFloat(valorCompra, 64)
		if err != nil {
			http.Error(w, "Invalid valorCompra", http.StatusBadRequest)
			return
		}

		valorVendaFloat, err := strconv.ParseFloat(valorVenda, 64)
		if err != nil {
			http.Error(w, "Invalid valorVenda", http.StatusBadRequest)
			return
		}

		produtoRef := firestoreClient.Client.Collection("produtos").Doc(id)

		// Atualiza os campos do documento no Firestore
		_, err = produtoRef.Set(firestoreClient.Ctx, map[string]interface{}{
			"NomeProduto": nomeProduto,
			"ValorCompra": valorCompraFloat,
			"ValorVenda":  valorVendaFloat,
		}, firestore.MergeAll)
		if err != nil {
			http.Error(w, "Failed to update product in Firestore", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/index", http.StatusSeeOther)
		return
	} else if r.Method == "GET" {
		vars := mux.Vars(r)
		id := vars["id"]

		produtoRef := firestoreClient.Client.Collection("produtos").Doc(id)

		// Recupera o documento do Firestore com o ID especificado
		snapshot, err := produtoRef.Get(firestoreClient.Ctx)
		if err != nil {
			http.Error(w, "Product not found in Firestore", http.StatusNotFound)
			return
		}

		var produto Produto
		if err := snapshot.DataTo(&produto); err != nil {
			http.Error(w, "Failed to parse product data", http.StatusInternalServerError)
			return
		}

		tmpl := template.Must(template.ParseFiles("template/edit.html"))
		data := ProdutoPageData{
			PageTitle: "Coffee Shop - Editar Produto",
			Produto:   produto,
		}

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		}
	} else {
		// Se o método não for POST ou GET, redirecionar para a lista de produtos
		http.Redirect(w, r, "/index", http.StatusSeeOther)
	}
}

func DeleteProdutoHandler(w http.ResponseWriter, r *http.Request) {

	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		vars := mux.Vars(r)
		id := vars["id"]

		produtoRef := firestoreClient.Client.Collection("produtos").Doc(id)

		// Deleta o documento do Firestore com o ID especificado
		_, err := produtoRef.Delete(firestoreClient.Ctx)
		if err != nil {
			http.Error(w, "Failed to delete product in Firestore", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/index", http.StatusSeeOther)
		return
	}

	// Se o método não for POST, redirecionar para a lista de produtos
	http.Redirect(w, r, "/index", http.StatusSeeOther)
}

func authenticate(w http.ResponseWriter, r *http.Request) bool {
	// Função de autenticação
	username, password, ok := r.BasicAuth()
	if !ok || username != "admin" || password != "coffeeShop40" {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	return true
}

func AbrirTicketHandler(w http.ResponseWriter, r *http.Request) {

	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		// Processar o formulário de abertura de ticket aqui
		titulo := r.FormValue("titulo")
		descricao := r.FormValue("descricao")

		novoTicket := Ticket{
			Titulo:       titulo,
			Descricao:    descricao,
			DataAbertura: time.Now(),
		}

		// Adicionar um novo documento à coleção "tickets" no Firestore
		_, _, err := firestoreClient.Client.Collection("tickets").Add(firestoreClient.Ctx, novoTicket)
		if err != nil {
			http.Error(w, "Failed to create ticket in Firestore", http.StatusInternalServerError)
			return
		}
	}

	// Recuperar a lista de tickets
	var tickets []Ticket

	ticketsRef, err := firestoreClient.Client.Collection("tickets").Documents(firestoreClient.Ctx).GetAll()
	if err != nil {
		http.Error(w, "Failed to fetch tickets from Firestore", http.StatusInternalServerError)
		return
	}

	for _, doc := range ticketsRef {
		var ticket Ticket
		if err := doc.DataTo(&ticket); err != nil {
			http.Error(w, "Failed to parse ticket data", http.StatusInternalServerError)
			return
		}
		tickets = append(tickets, ticket)
	}

	tmpl := template.Must(template.ParseFiles("template/abrir_ticket.html"))
	data := ProdutoPageData{
		PageTitle: "Coffee Shop - Abertura de Ticket",
		Tickets:   tickets,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func ListTicketsHandler(w http.ResponseWriter, r *http.Request) {

	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	// Recuperar a lista de tickets
	var tickets []Ticket

	ticketsRef, err := firestoreClient.Client.Collection("tickets").Documents(firestoreClient.Ctx).GetAll()
	if err != nil {
		http.Error(w, "Failed to fetch tickets from Firestore", http.StatusInternalServerError)
		return
	}

	for _, doc := range ticketsRef {
		var ticket Ticket
		if err := doc.DataTo(&ticket); err != nil {
			http.Error(w, "Failed to parse ticket data", http.StatusInternalServerError)
			return
		}
		tickets = append(tickets, ticket)
	}

	tmpl := template.Must(template.ParseFiles("template/tickets.html"))
	data := ProdutoPageData{
		PageTitle: "Coffee Shop - Lista de Tickets",
		Tickets:   tickets,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func ListProdutosHandler(w http.ResponseWriter, r *http.Request) {

	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	var produtos []Produto

	iter := firestoreClient.Client.Collection("produtos").Documents(firestoreClient.Ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, "Failed to fetch products from Firestore", http.StatusInternalServerError)
			return
		}
		var produto Produto
		if err := doc.DataTo(&produto); err != nil {
			http.Error(w, "Failed to parse product data", http.StatusInternalServerError)
			return
		}
		produtos = append(produtos, produto)
	}

	tmpl := template.Must(template.ParseFiles("template/index.html"))
	data := ProdutoPageData{
		PageTitle: "Coffee Shop - Manutenção de Estoque",
		Produtos:  produtos,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func VisualizarTransacoesHandler(w http.ResponseWriter, r *http.Request) {

	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	var transacoes []Transacao

	transacoesRef, err := firestoreClient.Client.Collection("transacoes").Documents(firestoreClient.Ctx).GetAll()
	if err != nil {
		http.Error(w, "Failed to fetch transactions from Firestore", http.StatusInternalServerError)
		return
	}

	for _, doc := range transacoesRef {
		var transacao Transacao
		if err := doc.DataTo(&transacao); err != nil {
			log.Printf("Failed to parse transaction data: %v", err)
			// Adicione um log para imprimir os dados do documento, se necessário
			log.Printf("Document data: %v", doc.Data())
			http.Error(w, "Failed to parse transaction data", http.StatusInternalServerError)
			return
		}
		transacoes = append(transacoes, transacao)
	}

	// Preparar os dados para o template
	data := TransacaoPageData{
		Transacoes: transacoes,
	}

	tmpl := template.Must(template.ParseFiles("template/visualizar_transacoes.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func RelatorioFluxoHandler(w http.ResponseWriter, r *http.Request) {
	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	// Consulta para buscar os meses e anos únicos das transações
	docs, err := firestoreClient.Client.Collection("transacoes").Documents(firestoreClient.Ctx).GetAll()
	if err != nil {
		http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
		return
	}

	// Usaremos um map para armazenar os meses e anos únicos
	uniqueDates := make(map[string]bool)
	for _, doc := range docs {
		data := doc.Data()
		dateInterface, exists := data["DataTransacao"]
		if !exists || dateInterface == nil {
			fmt.Printf("campo data_transacao não está presente ou é nulo\n")
			continue
		}

		date, ok := dateInterface.(time.Time)
		if !ok {
			fmt.Printf("campo data_transacao não é do tipo time.Time")
			continue
		}

		monthYear := fmt.Sprintf("%d-%d", date.Year(), date.Month())
		uniqueDates[monthYear] = true
	}

	var meses []string
	var anos []string
	for date := range uniqueDates {
		splitDate := strings.Split(date, "-")
		anos = append(anos, splitDate[0])
		meses = append(meses, splitDate[1])
	}

	data := RelatorioPageData{
		Meses: meses,
		Anos:  anos,
	}

	tmpl := template.Must(template.ParseFiles("template/relatorio_fluxo.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
		log.Printf("Failed to execute template: %v", err)
	}
}

func GerarRelatorioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {

		firestoreClient, err := InitializeFirestore()
		if err != nil {
			http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
			return
		}

		mesStr := r.FormValue("mes")
		anoStr := r.FormValue("ano")

		mes, err := strconv.Atoi(mesStr)
		if err != nil {
			http.Error(w, "Failed to convert month to integer", http.StatusBadRequest)
			return
		}

		ano, err := strconv.Atoi(anoStr)
		if err != nil {
			http.Error(w, "Failed to convert year to integer", http.StatusBadRequest)
			return
		}

		// Calcular o primeiro e último dia do mês no ano especificado
		firstDay := time.Date(ano, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
		lastDay := firstDay.AddDate(0, 1, -1).Add(24 * time.Hour)

		// Consultar as transações dentro do intervalo de datas
		docs, err := firestoreClient.Client.Collection("transacoes").Where("DataTransacao", ">=", firstDay).Where("DataTransacao", "<=", lastDay).Documents(firestoreClient.Ctx).GetAll()
		if err != nil {
			http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
			return
		}

		if len(docs) == 0 {
			http.Error(w, "No transactions found for the specified date range", http.StatusNotFound)
			return
		}

		// Criar um arquivo CSV
		fileName := fmt.Sprintf("./relatorios_fluxo/relatorio_%02d_%d.csv", mes, ano)
		file, err := os.Create(fileName)
		if err != nil {
			http.Error(w, "Failed to create CSV file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Escrever no arquivo CSV
		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Escrever cabeçalhos
		headers := []string{"ID", "CodigoTransacao", "CodigoProd", "NomeProd", "QuantidadeProd", "ValorTransacao", "DataTransacao"}
		if err := writer.Write(headers); err != nil {
			http.Error(w, "Failed to write CSV headers", http.StatusInternalServerError)
			return
		}

		// Escrever os dados das transações no arquivo CSV
		for _, doc := range docs {
			data := doc.Data()
			record := []string{
				doc.Ref.ID,
			}

			if val, ok := data["CodigoTransacao"].(int64); ok {
				record = append(record, strconv.Itoa(int(val)))
			} else {
				record = append(record, "null") // Tratamento para valor nulo ou não int64
			}

			if val, ok := data["CodigoProd"].(int64); ok {
				record = append(record, strconv.Itoa(int(val)))
			} else {
				record = append(record, "null") // Tratamento para valor nulo ou não int64
			}

			// Verificação e conversão para outros campos conforme necessário
			if val, ok := data["NomeProd"].(string); ok {
				record = append(record, val)
			} else {
				record = append(record, "null") // Tratamento para valor nulo ou não string
			}

			if val, ok := data["QuantidadeProd"].(int64); ok {
				record = append(record, strconv.Itoa(int(val)))
			} else {
				record = append(record, "null") // Tratamento para valor nulo ou não int64
			}

			if val, ok := data["ValorTransacao"].(float64); ok {
				record = append(record, fmt.Sprintf("%.2f", val))
			} else if val, ok := data["ValorTransacao"].(int64); ok {
				record = append(record, strconv.Itoa(int(val)))
			} else {
				record = append(record, "null") // Tratamento para valor nulo ou não int64
			}

			if val, ok := data["DataTransacao"].(time.Time); ok {
				record = append(record, val.Format("02/01/2006"))
			} else {
				record = append(record, "null") // Tratamento para valor nulo ou não time.Time
			}

			if err := writer.Write(record); err != nil {
				http.Error(w, "Failed to write CSV record", http.StatusInternalServerError)
				return
			}
		}

		// Redirecionar para a página de sucesso ou download do arquivo CSV
		fmt.Fprintf(w, `<html><body>Relatório gerado em: %s</body></html>`, fileName)
		fmt.Fprintf(w, `
			<script>
				setTimeout(function() {
					window.location.href = '/relatorio-fluxo';
				}, 5000); // Redirecionar após 5 segundos (5000 milissegundos)
			</script>
		`)
	}
}

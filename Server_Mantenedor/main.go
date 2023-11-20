package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

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

type Ticket struct {
	gorm.Model
	Titulo       string
	Descricao    string
	DataAbertura time.Time
}

type Transacao struct {
	gorm.Model
	CodigoTransacao uint
	CodigoProd      uint
	NomeProd        string
	QuantidadeProd  int
	ValorTransacao  float64
	DataTransacao   time.Time
	DataFormatted   string
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

var dbTickets *gorm.DB
var dbTransacoes *gorm.DB

func main() {
	db, err := gorm.Open(sqlite.Open("product.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect database")
	}
	db.AutoMigrate(&Produto{})

	dbTickets, err = gorm.Open(sqlite.Open("tickets.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to the tickets database")
	}
	dbTickets.AutoMigrate(&Ticket{})

	dbTransacoes, err = gorm.Open(sqlite.Open("transacoes.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to the tickets database")
	}
	dbTickets.AutoMigrate(&Transacao{})

	createAndPopulateTransacoesTable()

	// AutoMigrate para criar as tabelas no PostgreSQL
	db.AutoMigrate(&Produto{})
	db.AutoMigrate(&Ticket{})
	db.AutoMigrate(&Transacao{})

	createAndPopulateTransacoesTable()

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
	db, err := gorm.Open(sqlite.Open("product.db"), &gorm.Config{})
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
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

		produto := Produto{
			NomeProduto: nomeProduto,
			ValorCompra: valorCompraFloat,
			ValorVenda:  valorVendaFloat,
		}

		db.Create(&produto)

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

func EditProdutoHandler(w http.ResponseWriter, r *http.Request) {
	db, err := gorm.Open(sqlite.Open("product.db"), &gorm.Config{})
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

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

		var produto Produto
		if err := db.First(&produto, id).Error; err != nil {
			http.Error(w, "Produto não encontrado", http.StatusNotFound)
			return
		}

		produto.NomeProduto = nomeProduto
		produto.ValorCompra = valorCompraFloat
		produto.ValorVenda = valorVendaFloat

		db.Save(&produto)

		http.Redirect(w, r, "/index", http.StatusSeeOther)
		return
	} else if r.Method == "GET" {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		var produto Produto
		if err := db.First(&produto, id).Error; err != nil {
			http.Error(w, "Produto não encontrado", http.StatusNotFound)
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

	db, err := gorm.Open(sqlite.Open("product.db"), &gorm.Config{})
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}

	if r.Method == "POST" {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		var produto Produto
		if err := db.First(&produto, id).Error; err != nil {
			http.Error(w, "Produto não encontrado", http.StatusNotFound)
			return
		}

		db.Delete(&produto)

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
	if r.Method == "POST" {
		// Processar o formulário de abertura de ticket aqui
		titulo := r.FormValue("titulo")
		descricao := r.FormValue("descricao")

		novoTicket := Ticket{
			Titulo:       titulo,
			Descricao:    descricao,
			DataAbertura: time.Now(),
		}

		dbTickets.Create(&novoTicket)
	}

	// Recuperar a lista de tickets
	var tickets []Ticket
	dbTickets.Find(&tickets)

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
	// Recuperar a lista de tickets
	var tickets []Ticket
	dbTickets.Find(&tickets)

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

func VisualizarTransacoesHandler(w http.ResponseWriter, r *http.Request) {
	dbTransacoes, err := gorm.Open(sqlite.Open("transacoes.db"), &gorm.Config{})
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}

	var transacoes []Transacao
	dbTransacoes.Find(&transacoes)

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
	dbTransacoes, err := gorm.Open(sqlite.Open("transacoes.db"), &gorm.Config{})
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}

	var meses []string
	var anos []string

	// Consulta SQL para buscar os meses e anos únicos da coluna formatted_date
	var uniqueDates []struct {
		Mes string
		Ano string
	}
	if err := dbTransacoes.Raw("SELECT DISTINCT strftime('%m', data_transacao) as Mes, strftime('%Y', data_transacao) as Ano FROM transacaos").Scan(&uniqueDates).Error; err != nil {
		http.Error(w, "Failed to fetch unique dates", http.StatusInternalServerError)
		return
	}

	// Preencher as listas de meses e anos com os valores únicos
	for _, date := range uniqueDates {
		meses = append(meses, date.Mes)
		anos = append(anos, date.Ano)
	}

	data := RelatorioPageData{
		Meses: meses,
		Anos:  anos,
	}

	tmpl := template.Must(template.ParseFiles("template/relatorio_fluxo.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func GerarRelatorioHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		dbTransacoes, err := gorm.Open(sqlite.Open("transacoes.db"), &gorm.Config{})
		if err != nil {
			http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
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

		// Consulta SQL para buscar as transações dentro do intervalo de datas
		var transacoes []Transacao
		if err := dbTransacoes.Where("data_transacao BETWEEN ? AND ?", firstDay, lastDay).Find(&transacoes).Error; err != nil {
			http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
			return
		}

		if len(transacoes) == 0 {
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
		headers := []string{"ID", "CodigoTransacao", "CodigoProd", "NomeProd", "QuantidadeProd", "ValorTransacao", "DataTransacao", "FormattedDate"}
		if err := writer.Write(headers); err != nil {
			http.Error(w, "Failed to write CSV headers", http.StatusInternalServerError)
			return
		}

		// Escrever os dados das transações no arquivo CSV
		for _, transacao := range transacoes {
			record := []string{
				strconv.Itoa(int(transacao.ID)),
				strconv.Itoa(int(transacao.CodigoTransacao)),
				strconv.Itoa(int(transacao.CodigoProd)),
				transacao.NomeProd,
				strconv.Itoa(transacao.QuantidadeProd),
				fmt.Sprintf("%.2f", transacao.ValorTransacao),
				transacao.DataTransacao.Format("02/01/2006"),
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

func createAndPopulateTransacoesTable() {
	layout := "02/01/2006"

	db, err := gorm.Open(sqlite.Open("transacoes.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to the transacoes database")
	}

	// Migrate the schema
	err = db.AutoMigrate(&Transacao{})
	if err != nil {
		panic("Failed to migrate Transacao table")
	}

	var count int64
	db.Model(&Transacao{}).Count(&count)

	// Adiciona dados de exemplo apenas se a tabela estiver vazia
	if count == 0 {
		// Populate Transacao table with random data based on Produto
		var produtos []Produto
		dbProdutos, err := gorm.Open(sqlite.Open("product.db"), &gorm.Config{})
		if err != nil {
			panic("Failed to connect to the products database")
		}

		dbProdutos.Find(&produtos)

		// Create 20 random transactions for demonstration
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < 20; i++ {
			min := time.Date(2020, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
			max := time.Date(2023, time.Month(12)+1, 0, 0, 0, 0, 0, time.UTC)
			delta := max.Sub(min)
			randomTime := min.Add(time.Duration(rand.Int63n(int64(delta))))

			// Produto aleatório
			produto := produtos[rand.Intn(len(produtos))]
			quantidade := rand.Intn(3) + 1
			valorTransacao := float64(quantidade) * produto.ValorVenda
			valorTransacaoStr := fmt.Sprintf("%.2f", valorTransacao)
			valorTransacaoFloat, _ := strconv.ParseFloat(valorTransacaoStr, 64)
			dataTransacao := randomTime.Format(layout)
			dataTransacaoTime, err := time.Parse(layout, dataTransacao)
			if err != nil {
				fmt.Println("Erro ao converter string para time.Time:", err)
				return
			}

			transacao := Transacao{
				CodigoTransacao: uint(i + 1),
				CodigoProd:      produto.ID,
				NomeProd:        produto.NomeProduto,
				QuantidadeProd:  quantidade,
				ValorTransacao:  valorTransacaoFloat,
				DataTransacao:   dataTransacaoTime,
				DataFormatted:   dataTransacao,
			}

			db.Create(&transacao)
		}
	}
}

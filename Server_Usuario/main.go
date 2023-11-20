package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

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
	PageTitle          string
	Produtos           []Produto
	Produto            Produto
	ValorTotalCarrinho float64
	Carrinho           []CarrinhoItem
}

// Estrutura para os itens do carrinho
type CarrinhoItem struct {
	CodigoTransacao uint
	CodigoProduto   uint
	NomeProduto     string
	QuantidadeProd  uint
	ValorVenda      float64
	ValorTransacao  float64
	DataTransacao   string
}

var carrinho []CarrinhoItem // Estrutura de dados para o carrinho

func init() {
	// Inicializa a variável carrinho, se necessário
	carrinho = make([]CarrinhoItem, 0)
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
	http.HandleFunc("/adicionar-ao-carrinho", adicionarAoCarrinhoHandler)
	http.HandleFunc("/zerar_carrinho", zerarCarrinhoHandler)
	http.HandleFunc("/finalizar_compra", finalizarCompraHandler)

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
	// Defina os caminhos para o arquivo de origem e de destino
	origem := "C:/Users/albuq/go/src/PIT_II/Server_Mantenedor/product.db"
	destino := "C:/Users/albuq/go/src/PIT_II/Server_Usuario/product.db"

	// Abra o arquivo de origem para leitura
	arquivoOrigem, err := os.Open(origem)
	if err != nil {
		fmt.Println("Erro ao abrir o arquivo de origem:", err)
		return
	}
	defer arquivoOrigem.Close()

	// Cria ou sobrescreve o arquivo de destino
	arquivoDestino, err := os.Create(destino)
	if err != nil {
		fmt.Println("Erro ao criar o arquivo de destino:", err)
		return
	}
	defer arquivoDestino.Close()

	// Copia o conteúdo do arquivo de origem para o arquivo de destino
	_, err = io.Copy(arquivoDestino, arquivoOrigem)
	if err != nil {
		fmt.Println("Erro ao copiar o conteúdo do arquivo:", err)
		return
	}

	fmt.Println("Arquivo copiado com sucesso!")
	// Abre o banco de dados no diretório atual
	db, err := gorm.Open(sqlite.Open("product.db"), &gorm.Config{})
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
	// Calcula o valor total do carrinho
	valorTotalCarrinho := calcularValorTotalCarrinho()

	// Crie a estrutura de dados para enviar à página
	data := ProdutoPageData{
		PageTitle:          "Coffee Shop - Carrinho",
		ValorTotalCarrinho: valorTotalCarrinho, // Define o valor total do carrinho
		Carrinho:           carrinho,           // Passa os itens do carrinho para o template
	}

	// Carrega os dados na página HTML
	tmpl := template.Must(template.ParseFiles("template/carrinho.html"))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute template: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func adicionarAoCarrinhoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Obtenção dos dados do produto e quantidade do formulário enviado pelo front-end
	codigoProdutoStr := r.FormValue("codigoProduto")
	codigoProduto, err := strconv.ParseUint(codigoProdutoStr, 10, 64)
	if err != nil {
		http.Error(w, "Erro na conversão do código do produto", http.StatusBadRequest)
		return
	}
	nomeProduto := r.FormValue("nomeProduto")
	valorVenda, _ := strconv.ParseFloat(r.FormValue("valorVenda"), 64)
	quantidadeProd, _ := strconv.Atoi(r.FormValue("quantidadeProd"))

	// Obtenção do próximo código de transação
	proximoCodigoTransacao := proximoCodigoTransacao()

	// Criando um novo item do carrinho
	itemCarrinho := CarrinhoItem{
		CodigoTransacao: proximoCodigoTransacao,
		CodigoProduto:   uint(codigoProduto),
		NomeProduto:     nomeProduto,
		QuantidadeProd:  uint(quantidadeProd),
		ValorVenda:      valorVenda,
		ValorTransacao:  valorVenda * float64(quantidadeProd),
		DataTransacao:   time.Now().Format("2006-01-02"), // Formato da data: YYYY-MM-DD
	}

	// Adicionando o item ao carrinho
	carrinho = append(carrinho, itemCarrinho)

	// Responde ao front-end indicando sucesso
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Produto adicionado ao carrinho com sucesso!"))
}

// Função para calcular o próximo código de transação
func proximoCodigoTransacao() uint {
	if len(carrinho) == 0 {
		return 1
	}
	ultimoCodigo := carrinho[len(carrinho)-1].CodigoTransacao
	return ultimoCodigo + 1
}

func zerarCarrinhoHandler(w http.ResponseWriter, r *http.Request) {
	carrinho = make([]CarrinhoItem, 0) // Zera o carrinho
	fmt.Fprintln(w, "Carrinho zerado com sucesso!")
}

func finalizarCompraHandler(w http.ResponseWriter, r *http.Request) {
	// Aqui você pode incluir a lógica para adicionar as transações no banco de dados
	// e redirecionar o usuário para a página desejada após a compra
	// ...

	// Adicionar os itens do carrinho ao banco de dados (SQLite)
	db, err := gorm.Open(sqlite.Open("carrinho.db"), &gorm.Config{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect to database: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Migrate (cria) a tabela CarrinhoItem se ela ainda não existir
	if err := db.AutoMigrate(&CarrinhoItem{}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to migrate database: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Salvar os itens do carrinho no banco de dados
	for _, item := range carrinho {
		if err := db.Create(&item).Error; err != nil {
			http.Error(w, fmt.Sprintf("Failed to save item in database: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}

	// Limpar o carrinho após salvar os itens no banco de dados
	carrinho = make([]CarrinhoItem, 0)

	// Redirecionar o usuário para a página desejada após a compra
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Função para calcular o valor total do carrinho considerando a quantidade de cada item
func calcularValorTotalCarrinho() float64 {
	var valorTotal float64
	for _, item := range carrinho {
		valorTotal += float64(item.QuantidadeProd) * item.ValorVenda
	}
	return valorTotal
}

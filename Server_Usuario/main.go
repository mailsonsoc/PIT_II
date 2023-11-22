package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
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

type ProdutoPageData struct {
	PageTitle          string
	Produtos           []Produto
	Produto            Produto
	ValorTotalCarrinho float64
	Carrinho           []CarrinhoItem
}

// Estrutura para os itens do carrinho
type CarrinhoItem struct {
	CodigoTransacao int
	CodigoProduto   int
	NomeProduto     string
	QuantidadeProd  int
	ValorVenda      float64
	ValorTransacao  float64
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

// Handlers para cada endpoint
func paginaInicialHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "template/index.html")
}

func catalogoHandler(w http.ResponseWriter, r *http.Request) {
	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	// Realiza uma busca por todos os produtos na tabela
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

	// Inicializa o cliente Firestore
	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
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

	// Obtém o próximo código de transação baseado no próximo ID do documento na coleção "transacoes"
	proxCodigoTransacao, err := proximoCodigoTransacao(firestoreClient)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get next transaction ID: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Criando um novo item do carrinho
	itemCarrinho := CarrinhoItem{
		CodigoTransacao: proxCodigoTransacao,
		CodigoProduto:   int(codigoProduto),
		NomeProduto:     nomeProduto,
		QuantidadeProd:  int(quantidadeProd),
		ValorVenda:      valorVenda,
		ValorTransacao:  valorVenda * float64(quantidadeProd),
	}

	// Salva o item do carrinho na coleção "carrinho" no Firestore
	_, _, err = firestoreClient.Client.Collection("carrinho").Add(firestoreClient.Ctx, itemCarrinho)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save item in Firestore: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Adiciona o item ao slice carrinho local
	carrinho = append(carrinho, itemCarrinho)

	// Responde ao front-end indicando sucesso
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Produto adicionado ao carrinho com sucesso!"))
}

func proximoCodigoTransacao(firestoreClient *FirestoreClient) (int, error) {
	// Obtém o próximo código de transação baseado no próximo ID do documento na coleção "transacoes"
	iter := firestoreClient.Client.Collection("transacoes").Documents(firestoreClient.Ctx)
	numDocs := int(0)
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		numDocs++
	}
	return numDocs + 1, nil
}

func zerarCarrinhoHandler(w http.ResponseWriter, r *http.Request) {
	carrinho = make([]CarrinhoItem, 0) // Zera o carrinho
	fmt.Fprintln(w, "Carrinho zerado com sucesso!")
}

func finalizarCompraHandler(w http.ResponseWriter, r *http.Request) {
	// Inicializa o cliente Firestore
	firestoreClient, err := InitializeFirestore()
	if err != nil {
		http.Error(w, "Failed to connect to Firestore", http.StatusInternalServerError)
		return
	}

	// Salva os itens do carrinho na coleção "carrinho" no Firestore
	for _, item := range carrinho {
		itemData := map[string]interface{}{
			"codigo_transacao": item.CodigoTransacao,
			"codigo_produto":   item.CodigoProduto,
			"nome_produto":     item.NomeProduto,
			"quantidade_prod":  item.QuantidadeProd,
			"valor_venda":      item.ValorVenda,
			"valor_transacao":  item.ValorTransacao,
			"data_transacao":   time.Now(),
		}

		_, _, err := firestoreClient.Client.Collection("carrinho").Add(context.Background(), itemData)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save item in Firestore: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}

	// Salva os itens do carrinho na coleção "transacoes" no Firestore
	for _, item := range carrinho {
		itemData := map[string]interface{}{
			"codigo_transacao": item.CodigoTransacao,
			"codigo_produto":   item.CodigoProduto,
			"nome_produto":     item.NomeProduto,
			"quantidade_prod":  item.QuantidadeProd,
			"valor_venda":      item.ValorVenda,
			"valor_transacao":  item.ValorTransacao,
			"data_transacao":   time.Now(),
		}

		_, _, err := firestoreClient.Client.Collection("transacoes").Add(context.Background(), itemData)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save transaction in Firestore: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}

	// Exclui todos os documentos da coleção "carrinho" no Firestore, exceto o documento com ID "1"
	iter := firestoreClient.Client.Collection("carrinho").Documents(firestoreClient.Ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			http.Error(w, "Failed to fetch documents from Firestore", http.StatusInternalServerError)
			return
		}

		docID := doc.Ref.ID
		if docID != "1" {
			_, err := firestoreClient.Client.Collection("carrinho").Doc(docID).Delete(firestoreClient.Ctx)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to delete document in Firestore: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}
	}

	// Limpa o carrinho após salvar os itens no Firestore e excluir os itens do carrinho
	carrinho = make([]CarrinhoItem, 0)

	// Redireciona o usuário para a página desejada após a compra
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

package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"time"
)

type Block struct {
	Position     int
	Data         BookCheckout
	TimeStamp    string
	Hash         string
	PreviousHash string
}

type Book struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	PublishDate string `json:"publish_date"`
	ISBN        string `json:"isbn"`
}

type BookCheckout struct {
	BookID       string `json:"book_id"`
	User         string `json:"user"`
	CheckoutDate string `json:"checkout_date"`
	IsGenesis    bool   `json:"is_genesis"`
}

type Blockchain struct {
	blocks []*Block
}

var BlockChain *Blockchain

func (b *Block) generateHash() {
	dataByte, err := json.Marshal(b.Data)
	if err != nil {
		log.Fatalf("Errror to marshal block data : %v", err)
	}
	data := string(b.Position) + b.TimeStamp + string(dataByte) + b.PreviousHash

	hash := sha256.New()
	hash.Write([]byte(data))
	b.Hash = hex.EncodeToString(hash.Sum(nil))
}

func validBlock(block, prevBlock *Block) bool {
	if prevBlock.Hash != block.PreviousHash {
		return false
	}
	if !block.ValidateHash(block.Hash) {
		return false
	}

	if prevBlock.Position+1 != block.Position {
		return false
	}

	return true
}

func (b *Block) ValidateHash(hash string) bool {
	b.generateHash()
	if b.Hash != hash {
		return false
	}
	return true
}

func CreateBlock(prevBlock *Block, checkoutItem BookCheckout) *Block {
	block := &Block{}
	block.Position = prevBlock.Position + 1
	block.PreviousHash = prevBlock.Hash
	block.TimeStamp = time.Now().String()
	block.Data = checkoutItem
	block.generateHash()

	return block
}

func (bc *Blockchain) AddBlock(data BookCheckout) {
	prevBlock := bc.blocks[len(bc.blocks)-1]

	block := CreateBlock(prevBlock, data)

	if validBlock(block, prevBlock) {
		bc.blocks = append(bc.blocks, block)
	}

}

func newBook(w http.ResponseWriter, r *http.Request) {
	var book Book

	err := json.NewDecoder(r.Body).Decode(&book)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not create : %v", err)
		w.Write([]byte("Could not create new book"))
		return
	}

	h := md5.New()
	io.WriteString(h, book.ISBN+book.PublishDate)
	book.ID = fmt.Sprintf("%x", h.Sum(nil))

	resp, err := json.MarshalIndent(book, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not marshal paylod : %v", err)
		w.Write([]byte("Could not save book data"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func writeBlock(w http.ResponseWriter, r *http.Request) {
	var bookCheckout BookCheckout

	if err := json.NewDecoder(r.Body).Decode(&bookCheckout); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not write block : %v", err)
		w.Write([]byte("Could not write block"))
		return
	}

	BlockChain.AddBlock(bookCheckout)
	resp, err := json.MarshalIndent(bookCheckout, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not marshal payload : %v", err)
		w.Write([]byte("Could not write block"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, BookCheckout{IsGenesis: true})
}

func NewBlockChain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}

func getBlockchain(w http.ResponseWriter, r *http.Request) {
	byteData, err := json.MarshalIndent(BlockChain.blocks, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		log.Printf("Cannot unmarshall data : %v", err)
		return
	}
	io.WriteString(w, string(byteData))
}

func main() {

	BlockChain = NewBlockChain()
	r := mux.NewRouter()
	r.HandleFunc("/", getBlockchain).Methods("GET")
	r.HandleFunc("/", writeBlock).Methods("POST")
	r.HandleFunc("/new", newBook).Methods("POST")

	go func() {
		for _, block := range BlockChain.blocks {
			fmt.Printf("Prvious hash: %x\n", block.PreviousHash)
			bytesBlockChainData, _ := json.MarshalIndent(block.Data, "", " ")
			fmt.Printf("Data:%v\n", string(bytesBlockChainData))
			fmt.Printf("Hash:%x\n", block.Hash)
			fmt.Println()
		}
	}()

	log.Println("Listening on port 3000")
	http.ListenAndServe(":3000", r)
}

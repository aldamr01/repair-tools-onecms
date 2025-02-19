package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	args := os.Args
	fmt.Println(args)
	if len(args) <= 1 {
		fmt.Println("Not enough arguments")
		os.Exit(1)
	}
	fmt.Println("===== Running... =====")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30)*time.Second)
	defer cancel()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	DSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USERNAME"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"))
	dbClient, err := GetDBConnection(DSN)
	if err != nil {
		fmt.Println("❌ ERROR connecting to database")
		panic(err)
	}
	defer dbClient.Close()

	osIndex := os.Getenv("POST_INDEX")
	osClient, err := GetOSConnection(os.Getenv("OS_HOST"), os.Getenv("OS_USERNAME"), os.Getenv("OS_PASSWORD"))
	if err != nil {
		fmt.Println("❌ ERROR connecting to opensearch")
		panic(err)
	}

	onecmsDB := NewOneCMSDB(*dbClient)
	onecmsOS := NewOneCMSOS(osClient)

	if args[1] == "fix-url" {
		fmt.Println("🏃🏽‍➡️ Repairing post url...")
		if len(args) < 4 {
			panic("not enough argument")
		}

		err := fixURL(
			ctx,
			onecmsDB,
			onecmsOS,
			os.Args[2],
			os.Args[3],
			osIndex,
		)
		if err != nil {
			fmt.Printf("\nGot some errors\n----------------------\n %v", err)
		}
	}

	fmt.Println("\n✅ OK Done")
	os.Exit(0)
}

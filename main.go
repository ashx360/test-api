package main

import (
	"encoding/json"
	"log"
	"net/http"

	"test-api/database"
	"test-api/handlers"
	"test-api/repositories"
	"test-api/services"

	_ "github.com/lib/pq" // Add this line for PostgreSQL driver
	"github.com/spf13/viper"
)

type Config struct {
	Port   string `mapstructure:"PORT"`
	DBConn string `mapstructure:"DB_CONN"`
}

func main() {
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Warning: Could not read .env file: %v", err)
	}
	viper.AutomaticEnv()

	config := Config{
		Port:   viper.GetString("PORT"),
		DBConn: viper.GetString("DB_CONN"),
	}

	// Debug: Print config values
	log.Printf("Port: %s", config.Port)
	log.Printf("DBConn: %s", config.DBConn)

	if config.DBConn == "" {
		log.Fatal("DB_CONN environment variable is not set")
	}

	db, err := database.InitDB(config.DBConn)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	categoryRepo := repositories.NewCategoryRepository(db)
	categoryService := services.NewCategoryService(categoryRepo)
	categoryHandler := handlers.NewCategoryHandler(categoryService)

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "OK",
			"message": "API Running",
		})
	})
	http.HandleFunc("/api/categories/", categoryHandler.HandleCategories)

}

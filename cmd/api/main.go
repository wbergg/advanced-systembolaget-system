package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	systemetpoll "advanced-systembolaget-system"
	"advanced-systembolaget-system/internal/auth"
	"advanced-systembolaget-system/internal/db"
	"advanced-systembolaget-system/internal/handlers"
	"advanced-systembolaget-system/internal/systembolaget"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type AppConfig struct {
	AdminUser string `json:"admin_user"`
	AdminPass string `json:"admin_pass"`
	ListenIP  string `json:"listen_ip"`
	Port      string `json:"port"`
}

const (
	configFile = "config.json"
	dbFile     = "data/advanced-systembolaget-system.db"
)

func loadAppConfig(path string) (AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, err
	}
	if cfg.AdminUser == "" || cfg.AdminPass == "" {
		return AppConfig{}, fmt.Errorf("admin_user and admin_pass are required in %s", path)
	}
	if cfg.ListenIP == "" {
		cfg.ListenIP = "0.0.0.0"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return cfg, nil
}

func generateJWTSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate JWT secret: %v", err)
	}
	return hex.EncodeToString(b)
}

func main() {
	appCfg, err := loadAppConfig(configFile)
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	jwtSecret := generateJWTSecret()

	database, err := db.Open(dbFile)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	if err := database.SeedAdmin(appCfg.AdminUser, appCfg.AdminPass); err != nil {
		log.Fatalf("Failed to seed admin: %v", err)
	}

	authHandler := &handlers.AuthHandler{DB: database, JWTSecret: jwtSecret}
	userHandler := &handlers.UserHandler{DB: database}

	r := gin.Default()
	r.Use(cors.Default())

	api := r.Group("/api")
	{
		// Public
		api.POST("/login", authHandler.Login)

		// Authenticated
		authed := api.Group("/")
		authed.Use(auth.JWTMiddleware(jwtSecret))
		{
			authed.GET("/me", authHandler.Me)
			authed.PUT("/me/password", authHandler.ChangePassword)

			authed.GET("/products", listProducts(database))
			authed.GET("/products/distinct/:column", distinctValues(database))
			authed.GET("/products/:id", getProduct(database))
			authed.PATCH("/products/:id/notes", updateNote(database))
			authed.GET("/products/:id/comments", getComments(database))
			authed.POST("/products/:id/comments", addComment(database))
			authed.POST("/sync", syncProducts(database))
			authed.POST("/key/refresh", refreshKey())
			authed.GET("/key/status", keyStatus())

			authed.GET("/users/list", listAllUsers(database))

			authed.GET("/events", listEventsHandler(database))
			authed.POST("/events", createEventHandler(database))
			authed.GET("/events/:id", getEventHandler(database))
			authed.PATCH("/events/:id", updateEventHandler(database))
			authed.DELETE("/events/:id", deleteEventHandler(database))
			authed.PATCH("/events/:id/lock", setEventLockedHandler(database))
			authed.POST("/events/:id/invite", inviteToEventHandler(database))
			authed.DELETE("/events/:id/invite/:userId", uninviteFromEventHandler(database))
			authed.POST("/events/:id/import-list", importSharedListHandler(database))
			authed.POST("/events/:id/beers", addBeerToEventHandler(database))
			authed.DELETE("/events/:id/beers/:beerId", removeBeerFromEventHandler(database))
			authed.PUT("/events/:id/scores/:beerId", setScoreHandler(database))
			authed.DELETE("/events/:id/scores/:beerId", deleteScoreHandler(database))

			// Roll game
			authed.PATCH("/events/:id/public", toggleEventPublicHandler(database))
			authed.PATCH("/events/:id/visibility", setEventHiddenHandler(database))
			authed.GET("/events/:id/roll", getRollStateHandler(database))
			authed.POST("/events/:id/roll", performRollHandler(database))
			authed.POST("/events/:id/roll/reset", resetRollHandler(database))
			authed.DELETE("/events/:id/roll/pool/:poolId", undoConsumedHandler(database))
			authed.DELETE("/events/:id/roll/veto/:poolId", undoVetoHandler(database))
			authed.POST("/events/:id/roll/:turnId/accept", acceptRollHandler(database))
			authed.POST("/events/:id/roll/:turnId/veto", vetoRollHandler(database))

			// Shared lists
			authed.GET("/shared-lists", listSharedLists(database))
			authed.POST("/shared-lists", createSharedList(database))
			authed.GET("/shared-lists/:id", getSharedList(database))
			authed.PATCH("/shared-lists/:id", renameSharedList(database))
			authed.DELETE("/shared-lists/:id", deleteSharedList(database))
			authed.POST("/shared-lists/:id/items", addToSharedList(database))
			authed.PATCH("/shared-lists/:id/items/:productId", updateSharedListItem(database))
			authed.DELETE("/shared-lists/:id/items/:productId", removeFromSharedList(database))
			authed.PATCH("/shared-lists/:id/lock", setSharedListLocked(database))
			authed.POST("/shared-lists/:id/share", shareSharedList(database))
			authed.DELETE("/shared-lists/:id/share/:userId", unshareSharedList(database))
		}

		// Public shared list endpoint (no auth)
		api.GET("/public/shared-list/:uuid", getPublicSharedList(database))

		// Public roll endpoints (no auth)
		api.GET("/public/roll", getPublicRollHandler(database))
		api.POST("/public/roll", publicPerformRollHandler(database))
		api.POST("/public/roll/:turnId/accept", publicAcceptRollHandler(database))
		api.POST("/public/roll/:turnId/veto", publicVetoRollHandler(database))

		// Admin
		admin := api.Group("/admin")
		admin.Use(auth.JWTMiddleware(jwtSecret), auth.AdminOnly())
		{
			admin.GET("/users", userHandler.List)
			admin.POST("/users", userHandler.Create)
			admin.PUT("/users/:id", userHandler.Update)
			admin.DELETE("/users/:id", userHandler.Delete)
			admin.POST("/impersonate/:id", authHandler.Impersonate)
			admin.DELETE("/comments/:id", deleteComment(database))
			admin.DELETE("/products/:id", deleteProductHandler(database))
			admin.DELETE("/products", deleteAllProductsHandler(database))
		}
	}

	// Serve Vue frontend
	distFS, err := fs.Sub(systemetpoll.FrontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("Failed to get frontend dist: %v", err)
	}
	r.NoRoute(gin.WrapH(spaHandler(http.FS(distFS))))

	listenAddr := appCfg.ListenIP + ":" + appCfg.Port
	log.Printf("Starting server on %s", listenAddr)
	r.Run(listenAddr)
}

// spaHandler serves static files, falling back to index.html for SPA routes.
func spaHandler(fsys http.FileSystem) http.Handler {
	fileServer := http.FileServer(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		f, err := fsys.Open(r.URL.Path)
		if err != nil {
			// Fall back to index.html for SPA routing
			r.URL.Path = "/"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})
}

func listProducts(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		f := db.ListFilter{
			Search:   c.Query("search"),
			Category: c.Query("category"),
			SortBy:   c.DefaultQuery("sortBy", "name"),
			SortDir:  c.DefaultQuery("sortDir", "asc"),
			Name:     c.Query("name"),
			Producer: c.Query("producer"),
		}

		if v := c.Query("country"); v != "" {
			f.Countries = strings.Split(v, ",")
		}
		if v := c.Query("packaging"); v != "" {
			f.Packagings = strings.Split(v, ",")
		}
		if v := c.Query("volume"); v != "" {
			f.Volumes = strings.Split(v, ",")
		}
		if p := c.Query("page"); p != "" {
			f.Page, _ = strconv.Atoi(p)
		}
		if p := c.Query("pageSize"); p != "" {
			f.PageSize, _ = strconv.Atoi(p)
		}
		if v := c.Query("minPrice"); v != "" {
			p, _ := strconv.ParseFloat(v, 64)
			f.MinPrice = &p
		}
		if v := c.Query("maxPrice"); v != "" {
			p, _ := strconv.ParseFloat(v, 64)
			f.MaxPrice = &p
		}
		if v := c.Query("minAbv"); v != "" {
			p, _ := strconv.ParseFloat(v, 64)
			f.MinAbv = &p
		}
		if v := c.Query("maxAbv"); v != "" {
			p, _ := strconv.ParseFloat(v, 64)
			f.MaxAbv = &p
		}

		products, total, err := database.ListProducts(f)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"products": products,
			"total":    total,
			"page":     f.Page,
			"pageSize": f.PageSize,
		})
	}
}

func getProduct(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := database.GetProduct(c.Param("id"))
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

func distinctValues(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		vals, err := database.DistinctValues(c.Param("column"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, vals)
	}
}

func updateNote(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Note string `json:"note"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := database.SetNote(c.Param("id"), body.Note); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func getComments(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		comments, err := database.GetComments(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if comments == nil {
			comments = []db.Comment{}
		}
		c.JSON(http.StatusOK, comments)
	}
}

func addComment(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Comment string `json:"comment"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Comment == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "comment is required"})
			return
		}
		claims := auth.ClaimsFromContext(c)
		comment, err := database.AddComment(c.Param("id"), claims.UserID, body.Comment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, comment)
	}
}

func deleteComment(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
			return
		}
		if err := database.DeleteComment(id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func deleteProductHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := database.DeleteProduct(c.Param("id")); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func deleteAllProductsHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		n, err := database.DeleteAllProducts()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": n})
	}
}

func syncProducts(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Filters map[string]string `json:"filters"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}

		cfg, err := systembolaget.LoadConfig(configFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("no API key configured: %v", err)})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		query := systembolaget.BuildQueryFromMap(body.Filters)
		products, err := systembolaget.FetchAll(cfg.APIKey, query, func(page, totalPages, totalProducts int) {
			c.SSEvent("progress", gin.H{
				"page":       page,
				"totalPages": totalPages,
				"products":   totalProducts,
			})
			c.Writer.Flush()
		})
		if err != nil {
			c.SSEvent("error", gin.H{"error": err.Error()})
			c.Writer.Flush()
			return
		}

		if err := database.UpsertProducts(products); err != nil {
			c.SSEvent("error", gin.H{"error": fmt.Sprintf("db upsert failed: %v", err)})
			c.Writer.Flush()
			return
		}

		c.SSEvent("done", gin.H{"synced": len(products)})
		c.Writer.Flush()
	}
}

func refreshKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key, err := systembolaget.FetchAPIKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := systembolaget.SaveConfig(configFile, systembolaget.Config{APIKey: key}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func keyStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := systembolaget.LoadConfig(configFile)
		c.JSON(http.StatusOK, gin.H{
			"hasKey": err == nil && cfg.APIKey != "",
		})
	}
}

func listAllUsers(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := database.ListUsers()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Return only id + username (no passwords/roles)
		type safeUser struct {
			ID       int    `json:"userId"`
			Username string `json:"username"`
		}
		result := make([]safeUser, len(users))
		for i, u := range users {
			result[i] = safeUser{ID: u.ID, Username: u.Username}
		}
		c.JSON(http.StatusOK, result)
	}
}

// ── Event handlers ──

func listEventsHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		events, err := database.ListEvents(claims.UserID, claims.Role == "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if events == nil {
			events = []db.Event{}
		}
		c.JSON(http.StatusOK, events)
	}
}

func createEventHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			EventDate   string `json:"eventDate"`
			Type        string `json:"type"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if body.Type == "" {
			body.Type = "tasting"
		}
		if body.Type == "roll" && claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "only admins can create roll events"})
			return
		}
		ev, err := database.CreateEvent(body.Name, body.Description, body.EventDate, claims.UserID, body.Type)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, ev)
	}
}

func getEventHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		ev, err := database.GetEvent(id, claims.UserID, claims.Role == "admin")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		c.JSON(http.StatusOK, ev)
	}
}

func updateEventHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		var body struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			EventDate   string `json:"eventDate"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if err := database.UpdateEvent(id, body.Name, body.Description, body.EventDate, claims.UserID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func deleteEventHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		if err := database.DeleteEvent(id, claims.UserID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func setEventLockedHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		var body struct {
			Locked bool `json:"locked"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := database.SetEventLocked(id, body.Locked, claims.UserID, claims.Role == "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func inviteToEventHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		var body struct {
			UserID int `json:"userId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
			return
		}
		if err := database.InviteToEvent(id, claims.UserID, body.UserID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func uninviteFromEventHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		targetUserID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}
		if err := database.UninviteFromEvent(id, claims.UserID, targetUserID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func importSharedListHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		var body struct {
			ListID int `json:"listId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.ListID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "listId is required"})
			return
		}
		// Check if this is a roll event
		var eventType string
		if err := database.GetEventType(id, &eventType); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if eventType == "roll" {
			if err := database.ImportSharedListToRollPool(id, body.ListID, claims.UserID, claims.Role == "admin"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		} else {
			if err := database.ImportSharedListToEvent(id, body.ListID, claims.UserID, claims.Role == "admin"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func addBeerToEventHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		var body struct {
			ProductID string `json:"productId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.ProductID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "productId is required"})
			return
		}
		if err := database.AddBeerToEvent(id, body.ProductID, claims.UserID, claims.Role == "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func removeBeerFromEventHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		beerID, err := strconv.Atoi(c.Param("beerId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid beer id"})
			return
		}
		if err := database.RemoveBeerFromEvent(id, beerID, claims.UserID, claims.Role == "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func setScoreHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		beerID, err := strconv.Atoi(c.Param("beerId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid beer id"})
			return
		}
		var body struct {
			Score int `json:"score"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := database.SetScore(beerID, claims.UserID, body.Score); err != nil {
			status := http.StatusBadRequest
			if err.Error() == "event is locked" {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func deleteScoreHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		beerID, err := strconv.Atoi(c.Param("beerId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid beer id"})
			return
		}
		if err := database.DeleteScore(beerID, claims.UserID); err != nil {
			status := http.StatusBadRequest
			if err.Error() == "event is locked" {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ── Roll game handlers ──

func setEventHiddenHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		var body struct {
			Hidden bool `json:"hidden"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := database.SetEventHidden(id, body.Hidden, claims.UserID, claims.Role == "admin"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func getRollStateHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		_, err = database.CanAccessEvent(id, claims.UserID, claims.Role == "admin")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		state, err := database.GetRollState(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, state)
	}
}

func performRollHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		_, err = database.CanAccessEvent(id, claims.UserID, claims.Role == "admin")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		var body struct {
			UserID int `json:"userId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
			return
		}
		turn, err := database.PerformRoll(id, body.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, turn)
	}
}

func acceptRollHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		turnID, err := strconv.Atoi(c.Param("turnId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid turn id"})
			return
		}
		_, err = database.CanAccessEvent(id, claims.UserID, claims.Role == "admin")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		if err := database.AcceptRoll(id, turnID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func vetoRollHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		turnID, err := strconv.Atoi(c.Param("turnId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid turn id"})
			return
		}
		_, err = database.CanAccessEvent(id, claims.UserID, claims.Role == "admin")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		if err := database.VetoRoll(id, turnID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func undoVetoHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		poolID, err := strconv.Atoi(c.Param("poolId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pool id"})
			return
		}
		if err := database.UndoVeto(id, poolID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func undoConsumedHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		poolID, err := strconv.Atoi(c.Param("poolId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pool id"})
			return
		}
		if err := database.UndoConsumed(id, poolID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func resetRollHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		if err := database.ResetRoll(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// ── Shared list handlers ──

func listSharedLists(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		lists, err := database.ListSharedLists(claims.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if lists == nil {
			lists = []db.SharedList{}
		}
		c.JSON(http.StatusOK, lists)
	}
}

func createSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		l, err := database.CreateSharedList(body.Name, claims.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, l)
	}
}

func getSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		l, err := database.GetSharedList(id, claims.UserID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "list not found"})
			return
		}
		c.JSON(http.StatusOK, l)
	}
}

func deleteSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		if err := database.DeleteSharedList(id, claims.UserID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func renameSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if err := database.RenameSharedList(id, body.Name, claims.UserID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func setSharedListLocked(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		var body struct {
			Locked bool `json:"locked"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := database.SetSharedListLocked(id, body.Locked, claims.UserID, claims.Role == "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func updateSharedListItem(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		var body struct {
			Quantity int `json:"quantity"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := database.UpdateSharedListItemQuantity(id, c.Param("productId"), body.Quantity, claims.UserID); err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "list is locked" {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func addToSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		var body struct {
			ProductID string `json:"productId"`
			Quantity  int    `json:"quantity"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.ProductID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "productId is required"})
			return
		}
		if body.Quantity < 1 {
			body.Quantity = 1
		}
		if err := database.AddToSharedList(id, body.ProductID, body.Quantity, claims.UserID); err != nil {
			status := http.StatusBadRequest
			if err.Error() == "list is locked" {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func removeFromSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		if err := database.RemoveFromSharedList(id, c.Param("productId"), claims.UserID); err != nil {
			status := http.StatusNotFound
			if err.Error() == "list is locked" {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func shareSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		var body struct {
			UserID int `json:"userId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
			return
		}
		if err := database.ShareSharedList(id, claims.UserID, body.UserID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func unshareSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid list id"})
			return
		}
		targetUserID, err := strconv.Atoi(c.Param("userId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}
		if err := database.UnshareSharedList(id, claims.UserID, targetUserID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func getPublicSharedList(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		l, err := database.GetSharedListByUUID(c.Param("uuid"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "list not found"})
			return
		}
		c.JSON(http.StatusOK, l)
	}
}

func toggleEventPublicHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := auth.ClaimsFromContext(c)
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		isPublic, err := database.ToggleEventPublic(id, claims.UserID, claims.Role == "admin")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"public": isPublic})
	}
}

func getPublicRollHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ev, err := database.GetPublicEvent()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active public event"})
			return
		}
		state, err := database.GetRollState(ev.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		participants := []gin.H{{"userId": ev.OwnerID, "username": ev.OwnerName}}
		for _, a := range ev.Attendees {
			participants = append(participants, gin.H{"userId": a.UserID, "username": a.Username})
		}
		c.JSON(http.StatusOK, gin.H{
			"eventName":    ev.Name,
			"state":        state,
			"participants": participants,
		})
	}
}

func publicPerformRollHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ev, err := database.GetPublicEvent()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active public event"})
			return
		}
		var body struct {
			UserID int `json:"userId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
			return
		}
		turn, err := database.PerformRoll(ev.ID, body.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, turn)
	}
}

func publicAcceptRollHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ev, err := database.GetPublicEvent()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active public event"})
			return
		}
		turnID, err := strconv.Atoi(c.Param("turnId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid turn id"})
			return
		}
		if err := database.AcceptRoll(ev.ID, turnID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func publicVetoRollHandler(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ev, err := database.GetPublicEvent()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active public event"})
			return
		}
		turnID, err := strconv.Atoi(c.Param("turnId"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid turn id"})
			return
		}
		if err := database.VetoRoll(ev.ID, turnID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}


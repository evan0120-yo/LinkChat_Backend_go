package main

import (
	"context"
	"fmt"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/evan0120-yo/linkchat-go/internal/auth"
	"github.com/evan0120-yo/linkchat-go/internal/link"

	// 需要引入 firestore
	"cloud.google.com/go/firestore"
)

func clearCollection(ctx context.Context, client *firestore.Client, collectionName string) {
	fmt.Printf("[Drop] 正在清空 '%s' 集合...\n", collectionName)
	iter := client.Collection(collectionName).Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("讀取文件失敗 (%s): %v", collectionName, err)
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			log.Printf("刪除失敗 %s: %v", doc.Ref.ID, err)
		}
	}
}

func main() {
	// 1. 初始化資料庫
	os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8090")
	projectID := "dailo-467502"
	ctx := context.Background()
	conf := &firebase.Config{ProjectID: projectID}

	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		log.Fatalf("Firebase App 初始化失敗: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Firestore Client 建立失敗: %v", err)
	}
	defer client.Close()
	fmt.Println("Firestore (Local Emulator) 連線成功！ID:", projectID)

	// Create-Drop 邏輯 (重構後)
	// [新增] 也要清空 links 集合，避免開發髒資料
	clearCollection(ctx, client, "users")
	clearCollection(ctx, client, "link_users")
	clearCollection(ctx, client, "links")

	fmt.Println("資料庫已清空 (Drop Complete)")

	// ==========================================
	// 4. 依賴注入
	// ==========================================
	// [修改] 這裡回傳的是 Module Struct，改名比較清楚
	linkModule := link.NewLinkModule(client)

	// Auth 模組依賴 Link 的 CommandUseCase (用於 Sync)
	// 注意: 這裡從 linkModule 中取出 LinkUserCommandUseCase
	authHandler, authSeeder, authMiddleware, testHandler := auth.NewAuthModule(client, linkModule.LinkUserCommandUseCase)

	// ==========================================
	// 5. 啟動 Web Server
	// ==========================================
	r := gin.Default()

	// 基礎路由群組 /citrus
	rootGroup := r.Group("/citrus")

	rootGroup.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "up", "db": "firestore-emulator"})
	})

	// [註冊] Auth 路由
	authHandler.RegisterRoutes(rootGroup, authMiddleware)
	testHandler.RegisterRoutes(rootGroup, authMiddleware)

	// [註冊] Link 路由 (新增)
	// 這會掛載 /citrus/links/search, /citrus/links/apply 等
	linkModule.Handler.RegisterRoutes(rootGroup, authMiddleware)

	// ==========================================
	// 6. Seed
	// ==========================================
	if err := authSeeder.Seed(ctx); err != nil {
		log.Fatalf("資料初始化失敗: %v", err)
	}

	// Link Seeder
	if err := linkModule.Seeder.Seed(ctx); err != nil {
		log.Fatalf("Link 資料植入失敗: %v", err)
	}

	fmt.Println("Server is running at http://localhost:8082")
	if err := r.Run(":8082"); err != nil {
		log.Fatal("Server 啟動失敗:", err)
	}
}

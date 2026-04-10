package query

import (
	"context"
	"sort"
	"unicode"

	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"github.com/evan0120-yo/linkchat-go/internal/link/object/req"
	"github.com/evan0120-yo/linkchat-go/internal/link/object/resp"
	linkQuery "github.com/evan0120-yo/linkchat-go/internal/link/service/query"
)

type LinkQueryUseCase interface {
	SearchUsers(ctx context.Context, keyword string) ([]*model.LinkUser, error)
	// GetLinkList 取得好友列表 (包含篩選與排序)
	GetLinkList(ctx context.Context, userID string, filter req.ListLinkReq) ([]*resp.LinkItemResp, error)
	GetLinkedSubject(ctx context.Context, ownerID, subjectID string) (*model.LinkUser, error)
}

type linkQueryUseCase struct {
	userQueryService linkQuery.LinkUserQueryService
	linkQueryService linkQuery.LinkQueryService
}

func NewLinkQueryUseCase(
	userQueryService linkQuery.LinkUserQueryService,
	linkQueryService linkQuery.LinkQueryService,
) LinkQueryUseCase {
	return &linkQueryUseCase{
		userQueryService: userQueryService,
		linkQueryService: linkQueryService,
	}
}

func (uc *linkQueryUseCase) SearchUsers(ctx context.Context, keyword string) ([]*model.LinkUser, error) {
	return uc.userQueryService.SearchUsers(ctx, keyword)
}

func (uc *linkQueryUseCase) GetLinkedSubject(ctx context.Context, ownerID, subjectID string) (*model.LinkUser, error) {
	if ownerID == "" || subjectID == "" || ownerID == subjectID {
		return nil, nil
	}

	link, err := uc.linkQueryService.GetLinkByParticipants(ctx, ownerID, subjectID)
	if err != nil {
		return nil, err
	}
	if link == nil || link.Status != model.StatusActive {
		return nil, nil
	}

	subject, err := uc.userQueryService.GetLinkUserByID(ctx, subjectID)
	if err != nil {
		return nil, err
	}
	if subject == nil || !subject.IsActive {
		return nil, nil
	}

	return subject, nil
}

// GetLinkList 取得好友列表 (核心邏輯)
func (uc *linkQueryUseCase) GetLinkList(ctx context.Context, userID string, filter req.ListLinkReq) ([]*resp.LinkItemResp, error) {
	// 1. [撈取] 先找出所有與我有關的連結 (Links)
	links, err := uc.linkQueryService.GetLinksByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. [收集] 收集對方的 ID (準備批次查人)
	// 優化：預先分配 slice 容量，避免 append 時多次擴容
	targetIDs := make([]string, 0, len(links))
	for _, l := range links {
		// 判斷哪一個 ID 是對方
		otherID := l.TargetID
		if l.TargetID == userID {
			otherID = l.RequesterID
		}
		targetIDs = append(targetIDs, otherID)
	}

	// 3. [撈取] 批次查出對方詳細資料 (LinkUsers)
	users, err := uc.userQueryService.GetLinkUsersByIDs(ctx, targetIDs)
	if err != nil {
		return nil, err
	}

	// [優化] 轉成 Map 方便查找 (O(M))
	// 雖然有建置成本，但這能將後續查找複雜度從 O(N*M) 降為 O(N)
	userMap := make(map[string]*model.LinkUser)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 4. [組裝 & 過濾]
	var result []*resp.LinkItemResp

	for _, l := range links {
		// 4-1. 找出對方 ID
		otherID := l.TargetID
		if l.TargetID == userID {
			otherID = l.RequesterID
		}

		// 4-2. 找出對方資料 (若找不到可能代表資料不同步，暫時跳過)
		targetUser, exists := userMap[otherID]
		if !exists {
			continue
		}

		// 4-3. 判斷狀態與方向 (關鍵業務邏輯)

		// [修正 BUG]: 這裡必須強制轉型 string(l.Status)
		// 因為 l.Status 是 model.LinkStatus 型別，不轉型會導致後續無法賦值 string 字串
		status := string(l.Status)
		direction := "none"

		switch l.Status {
		case "pending":
			if l.RequesterID == userID {
				// 我加別人 -> 待申請 (Outgoing)
				status = "pending_sent"
				direction = "outgoing"
			} else {
				// 別人加我 -> 待同意 (Incoming)
				status = "pending_received"
				direction = "incoming"
			}
		case "rejected":
			status = "rejected"
			direction = "none"
		case "blocked":
			status = "blocked"
		case "active":
			status = "active"
			direction = "none"
		}

		// 4-4. 篩選器 (Filter)
		// filter.Filter: "all", "active", "received", "sent"
		wanted := false
		switch filter.Filter {
		case "active":
			if status == "active" {
				wanted = true
			}
		case "received":
			if status == "pending_received" {
				wanted = true
			}
		case "sent":
			if status == "pending_sent" {
				wanted = true
			}
		default: // "all" or empty
			if status != "blocked" { // 列表預設不顯示已封鎖
				wanted = true
			}
		}

		if wanted {
			result = append(result, &resp.LinkItemResp{
				LinkID:      l.ID,
				UserID:      targetUser.ID,
				DisplayName: targetUser.DisplayName,
				Status:      status,
				Direction:   direction,
			})
		}
	}

	// 5. [排序] Memory Sort
	// 規則1: 狀態權重 (好友 > 待同意 > 待申請)
	// 規則2: 名稱 (英文 > 中文 > 其他)
	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]

		// 5-1. 比較狀態權重
		weightA := getStatusWeight(a.Status)
		weightB := getStatusWeight(b.Status)

		if weightA != weightB {
			// 權重小的排前面 (1 > 2 > 3)
			return weightA < weightB
		}

		// 5-2. 比較名稱 (DisplayName)
		// 優先判斷是否為 ASCII (英文)
		isAsciiA := isASCII(a.DisplayName)
		isAsciiB := isASCII(b.DisplayName)

		// A 是英文, B 不是 -> A 排前面
		if isAsciiA && !isAsciiB {
			return true
		}
		// A 不是英文, B 是 -> B 排前面
		if !isAsciiA && isAsciiB {
			return false
		}

		// 同類別 (都是英文 或 都是中文)，直接比較字串 (Go 字串比較是逐 byte 比較，中文通常符合筆劃/Unicode序)
		return a.DisplayName < b.DisplayName
	})

	return result, nil
}

// Helper: 定義狀態權重
func getStatusWeight(status string) int {
	switch status {
	case "active":
		return 1 // 最優先
	case "pending_received":
		return 2 // 待同意次之
	case "pending_sent":
		return 3 // 待申請最後
	case "rejected":
		return 4 // 已拒絕放後面
	default:
		return 99
	}
}

// Helper: 判斷字串開頭是否為 ASCII (簡單判斷英文)
func isASCII(s string) bool {
	if len(s) == 0 {
		return false
	}
	// 取第一個 rune 判斷
	r := []rune(s)[0]
	return r <= unicode.MaxASCII
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package settings

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"video-in-chinese/server/app/gateway/internal/logic/settings"
	"video-in-chinese/server/app/gateway/internal/svc"
)

// Get current application settings
func GetSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := settings.NewGetSettingsLogic(r.Context(), svcCtx)
		resp, err := l.GetSettings()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

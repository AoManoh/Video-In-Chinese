// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package task

import (
	"net/http"

	"video-in-chinese/server/app/gateway/internal/logic/task"
	"video-in-chinese/server/app/gateway/internal/svc"
	"video-in-chinese/server/app/gateway/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Download task result file
func DownloadFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DownloadFileRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := task.NewDownloadFileLogic(r.Context(), svcCtx, w)
		err := l.DownloadFile(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		}
		// Note: No need to call httpx.Ok(w) because logic already wrote response
	}
}

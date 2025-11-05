// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package task

import (
	"net/http"

	"video-in-chinese/server/app/gateway/internal/logic/task"
	"video-in-chinese/server/app/gateway/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Upload a video file to create a new translation task
func UploadTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := task.NewUploadTaskLogic(r.Context(), svcCtx, r)
		resp, err := l.UploadTask()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

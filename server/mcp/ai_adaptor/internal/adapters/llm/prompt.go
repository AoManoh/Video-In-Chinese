package llm

import (
	"fmt"
	"strings"

	"video-in-chinese/server/mcp/ai_adaptor/internal/adapters"
)

func buildOptimizationPrompts(text string, ctx *adapters.OptimizationContext) (string, string) {
	var systemBuilder strings.Builder
	systemBuilder.WriteString("你是一名脚本优化编辑，需要在不改变原意的前提下调整句子，使其便于口语配音。\n")
	systemBuilder.WriteString("请遵循以下规则：\n")
	if ctx != nil && ctx.SpeakerRole != "" {
		systemBuilder.WriteString(fmt.Sprintf("- 说话人角色：%s\n", ctx.SpeakerRole))
	}
	if ctx != nil && ctx.VideoType != "" {
		systemBuilder.WriteString(fmt.Sprintf("- 视频风格：%s\n", ctx.VideoType))
	}
	if ctx != nil && ctx.TargetDurationSeconds > 0 {
		systemBuilder.WriteString(fmt.Sprintf("- 朗读时长需接近 %.2f 秒\n", ctx.TargetDurationSeconds))
	}
	if ctx != nil && ctx.TargetWordMin > 0 && ctx.TargetWordMax > 0 {
		systemBuilder.WriteString(fmt.Sprintf("- 控制字数在 %d-%d 字之间\n", ctx.TargetWordMin, ctx.TargetWordMax))
	}
	systemBuilder.WriteString("- 仅输出优化后的文本，不要提供解释。")

	var userBuilder strings.Builder
	userBuilder.WriteString("请根据以上要求优化以下文本：\n\n")
	userBuilder.WriteString(text)

	return systemBuilder.String(), userBuilder.String()
}

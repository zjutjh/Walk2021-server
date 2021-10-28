package controller

import (
	"github.com/gin-gonic/gin"
	"walk-server/model"
	"walk-server/utility"
	"walk-server/utility/initial"
)

func GetInfo(context *gin.Context) {
	// 获取 open ID
	jwtToken := context.GetHeader("Authorization")[7:]
	jwtData, _ := utility.ParseToken(jwtToken) // 中间件校验过数据了
	openID := jwtData.OpenID

	// 获取用户数据
	person := model.Person{}
	initial.DB.Where("open_id = ?", openID).First(&person)

	utility.ResponseSuccess(context, gin.H{
		"name":      person.Name,
		"stu_id":    person.StuId,
		"gender":    person.Gender,
		"campus":    person.Campus,
		"create_op": person.CreatedOp,
		"join_op":   person.JoinOp,
		"team_id":   person.TeamId,
		"contact": gin.H{
			"qq":     person.Qq,
			"wechat": person.Wechat,
			"tel":    person.Tel,
		},
	})
}

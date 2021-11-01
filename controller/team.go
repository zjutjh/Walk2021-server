package controller

import (
	"github.com/gin-gonic/gin"
	"walk-server/model"
	"walk-server/utility"
	"walk-server/utility/initial"
)

// CreateTeamData 接收创建团队信息的数据类型
type CreateTeamData struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
	Route    uint8  `json:"route" binding:"required"`
}

// UpdateTeamData 更新团队信息的数据类型
type UpdateTeamData struct {
	Name  string `json:"name"`
	Route uint8  `json:"route"`
}

// JoinTeamData 加入团队时接收的信息类型
type JoinTeamData struct {
	TeamID   int    `json:"team_id"`
	Password string `json:"password"`
}

func CreateTeam(context *gin.Context) {
	// TODO 加入对当天团队数上限的判断
	// 获取 jwt 数据
	jwtToken := context.GetHeader("Authorization")[7:]
	jwtData, _ := utility.ParseToken(jwtToken)

	// 获取 post json 数据
	var createTeamData CreateTeamData
	err := context.ShouldBindJSON(&createTeamData)
	if err != nil {
		utility.ResponseError(context, "参数错误")
		return
	}

	// 查询用户信息
	openID := jwtData.OpenID
	var person model.Person
	initial.DB.Where("open_id = ?", openID).First(&person)

	if person.Status != 0 { // 现在已经加入了一个团队
		utility.ResponseError(context, "请先退出或解散原来的团队")
	}

	if person.CreatedOp == 0 {
		utility.ResponseError(context, "无法创建团队了")
	} else {
		// 再数据库中插入一个团队
		team := model.Team{
			Name:      createTeamData.Name,
			Num:       1,
			Password:  createTeamData.Password,
			Captain:   person.Name,
			Route:     createTeamData.Route,
			Submitted: false,
		}
		initial.DB.Create(&team)

		// 将入团队后对应的状态更新
		person.CreatedOp -= 1
		person.Status = 1

		initial.DB.Model(&person).Updates(person) // 将新的用户信息写入数据库

		// 返回新的 team_id 和 jwt 数据
		utility.ResponseSuccess(context, gin.H{
			"team_id": team.ID,
		})
	}
}

func JoinTeam(context *gin.Context) {
	// 获取 jwt 数据
	jwtToken := context.GetHeader("Authorization")[7:]
	jwtData, _ := utility.ParseToken(jwtToken)

	// 从数据库中读取用户信息
	var person model.Person
	initial.DB.Where("open_id = ?", jwtData.OpenID).Find(&person)

	if person.Status != 0 { // 如果在一个团队中
		utility.ResponseError(context, "请退出或解散原来的团队")
		return
	}

	var joinTeamData JoinTeamData
	err := context.ShouldBindJSON(&joinTeamData)
	if err != nil { // 参数发送错误
		utility.ResponseError(context, "参数错误")
		return
	}

	if person.JoinOp == 0 { // 加入次数用完了
		utility.ResponseError(context, "没有加入次数了")
		return
	}

	// 检查密码
	var team model.Team
	result := initial.DB.Where("id = ?", joinTeamData.TeamID).First(&team)
	if result.RowsAffected == 0 {
		utility.ResponseError(context, "找不到团队")
		return
	}
	if team.Submitted == true {
		utility.ResponseError(context, "该队伍已提交，无法加入")
		return
	}
	if team.Password != joinTeamData.Password {
		utility.ResponseError(context, "密码错误")
		return
	}

	// 如果人数没有大于团队最大人数
	result = initial.DB.Model(&team).Where("num < 6").Update("num", team.Num+1) // 队伍上限 6 人
	if result.RowsAffected == 0 {
		utility.ResponseError(context, "队伍人数到达上限")
	} else {
		person.Status = 1
		person.JoinOp--
		initial.DB.Model(&person).Updates(person) // 将新的用户信息写入数据库
		utility.ResponseSuccess(context, nil)
	}
}

func GetTeamInfo(context *gin.Context) {
	// 获取 jwt 数据
	jwtToken := context.GetHeader("Authorization")[7:]
	jwtData, _ := utility.ParseToken(jwtToken)

	// 获取个人信息
	var person model.Person
	initial.DB.Where("open_id = ?", jwtData.OpenID).First(&person)

	// 先判断是否加入了团队
	if person.Status == 0 {
		utility.ResponseError(context, "尚未加入团队")
		return
	}

	// 查找团队
	var team model.Team
	initial.DB.Where("id = ?", person.TeamId).First(&team)

	// 查找团队成员
	var persons []model.Person
	var leader model.Person
	var members []gin.H
	initial.DB.Where("team_id = ?", person.TeamId).First(&persons)
	for _, person := range persons {
		if person.Status == 2 { // 队长
			leader = person
		} else {
			members = append(members, gin.H{
				"name":   person.Name,
				"gender": person.Gender,
				"contact": gin.H{
					"qq":     person.Qq,
					"wechat": person.Wechat,
					"tel":    person.Tel,
				},
			})
		}
	}

	// 返回结果
	utility.ResponseSuccess(context, gin.H{
		"id":    person.TeamId,
		"name":  team.Name,
		"route": team.Route,
		"leader": gin.H{
			"name":   leader.Name,
			"gender": leader.Gender,
			"contact": gin.H{
				"qq":     leader.Qq,
				"wechat": leader.Wechat,
				"tel":    leader.Tel,
			},
		},
		"member": members,
	})
}

func DisbandTeam(context *gin.Context) {
	// 获取 jwt 数据
	jwtToken := context.GetHeader("Authorization")[7:]
	jwtData, _ := utility.ParseToken(jwtToken)

	// 查找用户
	var person model.Person
	initial.DB.Where("open_id = ?", jwtData.OpenID).First(&person)

	if person.Status == 0 {
		utility.ResponseError(context, "请先创建一个队伍")
		return
	} else if person.Status == 1 {
		utility.ResponseError(context, "队员无法解散队伍")
		return
	}

	// 查找团队
	var team model.Team
	initial.DB.Where("id = ?", person.TeamId).First(&team)

	// 查找团队所有用户
	var persons []model.Person
	initial.DB.Where("team_id = ?", person.TeamId).First(&persons)

	// 删除团队记录
	initial.DB.Delete(&team)

	// 还原所有队员的权限和所属团队ID
	for _, person := range persons {
		person.Status = 0
		person.TeamId = -1
		initial.DB.Save(&person)
	}
}

func LeaveTeam(context *gin.Context) {
	// 获取 jwt 数据
	jwtToken := context.GetHeader("Authorization")[7:]
	jwtData, _ := utility.ParseToken(jwtToken)

	// 查找用户
	var person model.Person
	initial.DB.Where("open_id = ?", jwtData.OpenID).First(&person)

	if person.Status == 0 {
		utility.ResponseError(context, "请先加入队伍")
		return
	} else if person.Status == 2 {
		utility.ResponseError(context, "队长只能解散队伍")
		return
	}

	// 恢复队员信息到未加入的状态
	person.Status = 0
	person.TeamId = -1
	initial.DB.Save(&person)

	utility.ResponseSuccess(context, nil)
}

func RemoveMember(context *gin.Context) {
	// 获取 jwt 数据
	jwtToken := context.GetHeader("Authorization")[7:]
	jwtData, _ := utility.ParseToken(jwtToken)

	// 查找用户
	var person model.Person
	initial.DB.Where("open_id = ?", jwtData.OpenID).First(&person)

	if person.Status == 0 {
		utility.ResponseError(context, "请先加入团队")
		return
	} else if person.Status == 1 {
		utility.ResponseError(context, "只有队长可以移除队员")
		return
	}

	// 读取 Get 参数
	memberRemovedOpenID := context.Query("openid")

	var personRemoved model.Person
	result := initial.DB.Where("open_id = ?", memberRemovedOpenID).First(personRemoved)
	if result.RowsAffected == 0 {
		utility.ResponseError(context, "没有这个用户")
		return
	}

	// 更新被踢出的人的状态
	personRemoved.Status = 0
	personRemoved.TeamId = -1
	initial.DB.Save(&personRemoved)
}

func UpdateTeam(context *gin.Context) {
	// 获取 jwt 数据
	jwtToken := context.GetHeader("Authorization")[7:]
	jwtData, _ := utility.ParseToken(jwtToken)

	// 查找用户
	var person model.Person
	initial.DB.Where("open_id = ?", jwtData.OpenID).First(&person)

	// 判断用户权限
	if person.Status == 0 {
		utility.ResponseError(context, "请先加入队伍")
		return
	} else if person.Status == 1 {
		utility.ResponseError(context, "没有修改的权限")
		return
	}

	// 解析 post 数据
	var updateTeamData UpdateTeamData
	err := context.ShouldBindJSON(&updateTeamData)
	if err != nil {
		utility.ResponseError(context, "参数错误")
		return
	}

	// 更新团队信息
	var team model.Team
	initial.DB.Where("team_id = ?", person.TeamId).First(&team)
	team.Name = updateTeamData.Name
	team.Route = updateTeamData.Route
	initial.DB.Save(&team)
}
package controllers

import (
	"github.com/astaxie/beego"
	"github.com/dcosapp/gocmdb/server/controllers/auth"
	"github.com/dcosapp/gocmdb/server/models"
	"net/http"
)

type HomePageController struct {
	LayoutController
}

// homepage/index
func (c *HomePageController) Index() {
	c.Redirect(beego.URLFor(beego.AppConfig.String("home")), http.StatusFound)
}

type DashboardPageController struct {
	LayoutController
}

// dashboardpage/index
func (c *DashboardPageController) Index() {
	c.Data["menu"] = "dashboard"
	c.TplName = "dashboard_page/index.html"
	c.LayoutSections["LayoutScript"] = "dashboard_page/index.script.html"
	c.Data["online"], c.Data["offline"], c.Data["undealed"] = models.DefaultAlarmManager.Dashboard()
}

type DashboardController struct {
	auth.LoginRequireController
}

func (c *DashboardController) Stat() {
	online, offline, alarmCount := models.DefaultAlarmManager.Dashboard()
	days, data := models.DefaultAlarmManager.GetLastestNStat(7)
	c.Data["json"] = map[string]interface{}{
		"code": 200,
		"text": "获取成功",
		"result": map[string]interface{}{
			"agent_offline_count": offline,
			"alarm_online_count":  online,
			"alarm_count":         alarmCount,
			"alarm_dist":          models.DefaultAlarmManager.GetStatForNotComplete(),
			"alarm_trend": map[string]interface{}{
				"days": days,
				"data": data,
			},
		},
	}
	c.ServeJSON()
}

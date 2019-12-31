package models

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"strconv"
	"time"
)

type Alarm struct {
	Id           int        `orm:"column(id);"json:"id"`
	Name         string     `orm:"column(name);size(64);null;"json:"name"`
	UUID         string     `orm:"column(uuid);size(64);null;"json:"uuid"`
	Type         int        `orm:"column(type);"json:"type"`
	Status       int        `orm:"column(status);"json:"status"`
	Reason       string     `orm:"column(reason);size(1024);null;"json:"reason"`
	Desc         string     `orm:"column(desc);size(1024);null;"json:"desc"`
	CreatedTime  *time.Time `orm:"column(created_time);auto_now_add;"json:"created_time"`
	DealedTime   *time.Time `orm:"column(dealed_time);null;"json:"dealed_time"`
	CompleteTime *time.Time `orm:"column(complete_time);null;"json:"complete_time"`

	// created_time 产生时间
	// uuid 终端
	// type 类型（1:离线 2：cpu 3：内存）
	// description 告警描述
	// status 状态(0:未处理, 1:正在处理， 2:已处理)
	// dealed_time 处理时间
	// reason 告警原因说明
	// notices 通知方式(sms email)
	// user 通知给谁
}

type AlarmManager struct {
}

func NewAlarmManager() *AlarmManager {
	return &AlarmManager{}
}

func (m *AlarmManager) CreateAlarm(typ int, uuid []orm.Params, reason string) {
	//port, _ := beego.AppConfig.Int("smtp::port")
	//host := beego.AppConfig.String("smtp::host")
	//user := beego.AppConfig.String("smtp::user")
	//password := beego.AppConfig.String("smtp::password")
	//emailSender := utils.NewEmail(host, port, user, password)
	//to := beego.AppConfig.Strings("smtp::to")
	//err := emailSender.Send(to, "[CMDB]终端离线告警", fmt.Sprintf("终端离线时间超过了%d分钟", alarm.Time), []string{})
	//beego.Info("[CMDB]终端离线告警", fmt.Sprintf("终端离线时间超过了%d分钟", alarm.Time), err)
	noticeWindows := 60
	noticeCounter := int64(2)
	noticeStartTime := time.Now().Add(-time.Duration(noticeWindows) * time.Minute)
	ormer := orm.NewOrm()
	for _, value := range uuid {
		id := fmt.Sprintf("%s", value["uuid"])
		alarmCnt := DefaultAlarmManager.GetCountByUuidAndType(id, typ, noticeStartTime)

		if alarmCnt > noticeCounter {
			//beego.Info(fmt.Printf("通知次数(%d)超过限制(%d)", alarmCnt, noticeCounter))
			continue
		}
		_, _ = ormer.Insert(&Alarm{
			Name:   DefaultAgentManager.GetByUUID(id).Name,
			UUID:   id,
			Type:   typ,
			Status: 0,
			Reason: reason,
		})
	}
}

func (m *AlarmManager) GetCountByUuidAndType(id string, typ int, noticeStartTime time.Time) int64 {
	cnt, _ := orm.NewOrm().QueryTable(&Alarm{}).Filter("uuid__exact", id).Filter("type__exact", typ).Filter("created_time__gte", noticeStartTime).Count()
	return cnt
}

func (m *AlarmManager) GetNotification(limit int) (int64, []*Alarm) {
	ormer := orm.NewOrm()
	queryset := ormer.QueryTable(&Alarm{})

	cnt, _ := queryset.Filter("status__exact", AlarmStatusNew).Count()

	var result []*Alarm
	queryset.Filter("status__exact", AlarmStatusNew).OrderBy("-created_time").Limit(limit).All(&result)
	return cnt, result
}

func (m *AlarmManager) Query(q string, start int64, length int) ([]*Alarm, int64, int64) {
	ormer := orm.NewOrm()
	queryset := ormer.QueryTable(&Alarm{})
	condition := orm.NewCondition()
	//condition = condition.And("deleted_time__isnull", true)
	total, _ := queryset.SetCond(condition).Count()

	qtotal := total
	if q != "" {
		query := orm.NewCondition()
		query = query.Or("name__icontains", q)
		query = query.Or("uuid__icontains", q)
		query = query.Or("reason__icontains", q)
		query = query.Or("desc__icontains", q)

		condition = condition.AndCond(query)

		qtotal, _ = queryset.SetCond(condition).Count()
	}
	var result []*Alarm
	_, _ = queryset.SetCond(condition).RelatedSel().Limit(length).Offset(start).OrderBy("-id").All(&result)
	return result, total, qtotal
}

func (m *AlarmManager) Dashboard() (online, offline, undealed int64) {
	alarm := DefaultAlarmSettingManager.GetById(1)
	endTime := time.Now().Add(-time.Duration(alarm.Time) * time.Minute)

	ormer := orm.NewOrm()
	online, _ = ormer.QueryTable(&Agent{}).Filter("heartbeat_time__gte", endTime).Count()
	offline, _ = ormer.QueryTable(&Agent{}).Filter("heartbeat_time__lt", endTime).Count()
	undealed, _ = ormer.QueryTable(&Alarm{}).Exclude("status__exact", AlarmStatusComplete).Count()
	return online, offline, undealed
}

func (m *AlarmManager) GetStatForNotComplete() map[string]int64 {
	var lines []orm.Params
	orm.NewOrm().Raw("select type, count(*) as cnt from alarm where status in (?,?) group by type", []int{AlarmStatusNew, AlarmStatusDoing}).Values(&lines)

	result := map[string]int64{}
	for _, line := range lines {
		typ := line["type"].(string)
		cntString := line["cnt"].(string)
		cnt, _ := strconv.ParseInt(cntString, 10, 64)
		result[typ] = cnt
	}
	return result
}

func (m *AlarmManager) GetLastestNStat(day int) ([]string, map[string][]int64) {
	endTime := time.Now()
	startTime := endTime.Add(-24*time.Duration(day-1)*time.Hour - 1)
	var lines []orm.Params

	orm.NewOrm().Raw("select date_format(created_time, '%Y-%m-%d') as day, type, status, count(*) as cnt from alarm where created_time >= ? group by day, type, status", startTime).Values(&lines)

	fmt.Println(lines)
	tempStat := map[string]map[string]int64{}

	for _, line := range lines {
		day, _ := line["day"].(string)
		status, _ := line["status"].(string)
		typ, _ := line["type"].(string)
		cntString, _ := line["cnt"].(string)
		cnt, _ := strconv.ParseInt(cntString, 10., 64)
		key := fmt.Sprintf("%s-%s", typ, status)

		if _, ok := tempStat[key]; !ok {
			tempStat[key] = map[string]int64{}
		}
		tempStat[key][day] = cnt
	}

	days := []string{}
	result := map[string][]int64{}

	for startTime.Before(endTime) {
		day := startTime.Format("2006-01-02")
		days = append(days, day)

		for _, typ := range []int{AlarmStatusNew, AlarmStatusDoing, AlarmStatusComplete} {
			for _, status := range []int{AlarmTypeCPU, AlarmTypeOffline, AlarmTypeRam} {
				key := fmt.Sprintf("%d-%d", typ, status)
				value := int64(0)
				if stat, ok := tempStat[key]; ok {
					value = stat[day]
				} else {
				}
				result[key] = append(result[key], value)
			}
		}
		startTime = startTime.Add(24 * time.Hour)
	}
	return days, result
}

func (m *AlarmManager) SetStatusById(id, status int) error {
	if status == 1 {
		_, err := orm.NewOrm().QueryTable(&Alarm{}).Filter("id__exact", id).Update(orm.Params{"status": status, "dealed_time": time.Now()})
		return err
	}
	_, err := orm.NewOrm().QueryTable(&Alarm{}).Filter("id__exact", id).Update(orm.Params{"status": status, "complete_time": time.Now()})
	return err
}

type AlarmSetting struct {
	Id        int    `orm:"column(id);"json:"id"`
	Name      string `orm:"column(name);size(64);null;"json:"name"`
	Type      int    `orm:"column(type);"json:"type"`
	Time      int    `orm:"column(time);"json:"time"`
	Threshold int    `orm:"column(threshold);"json:"threshold"`
	Counter   int    `orm:"column(counter);"json:"counter"`
}

type AlarmSettingManager struct {
}

func NewAlarmSettingManager() *AlarmSettingManager {
	return &AlarmSettingManager{}
}

func (m *AlarmSettingManager) Query(q string, start int64, length int) ([]*AlarmSetting, int64, int64) {
	ormer := orm.NewOrm()
	queryset := ormer.QueryTable(&AlarmSetting{})
	condition := orm.NewCondition()
	total, _ := queryset.SetCond(condition).Count()

	qtotal := total
	if q != "" {
		query := orm.NewCondition()
		query = query.Or("name__icontains", q)
		query = query.Or("type__icontains", q)
		condition = condition.AndCond(query)

		qtotal, _ = queryset.SetCond(condition).Count()
	}
	var result []*AlarmSetting
	_, _ = queryset.SetCond(condition).RelatedSel().Limit(length).Offset(start).All(&result)
	return result, total, qtotal
}

func (m *AlarmSettingManager) GetById(id int) *AlarmSetting {
	var set = &AlarmSetting{Id: id}
	_ = orm.NewOrm().QueryTable(set).Filter("Id__exact", id).One(set)
	return set
}

func (m *AlarmSettingManager) Modify(id, time, threshold, counter int) (*AlarmSetting, error) {
	set := &AlarmSetting{Id: id}
	ormer := orm.NewOrm()
	err := ormer.Read(set)
	if err == nil {
		set.Time = time
		set.Threshold = threshold
		set.Counter = counter
		if _, err := ormer.Update(set); err == nil {
			return set, nil
		}
	}
	return &AlarmSetting{}, err
}

var DefaultAlarmManager = NewAlarmManager()
var DefaultAlarmSettingManager = NewAlarmSettingManager()

func init() {
	orm.RegisterModel(&Alarm{}, &AlarmSetting{})
}

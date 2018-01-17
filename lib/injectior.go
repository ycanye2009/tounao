package lib

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"io/ioutil"
	"strings"
	"strconv"
	"github.com/coreos/goproxy"
	"fmt"
	"tounao/util"
	"time"
)

var questions = make(map[string]Question)
var tapTask *time.Ticker
var restartTask *time.Timer

func Injection(bytes []byte, ctx *goproxy.ProxyCtx) (data []byte) {

	data = bytes
	content := string(bytes)

	//log.Printf("path:%s\n content:%s", ctx.Req.URL.Path, content)

	if strings.Contains(content, "roomID") && strings.Contains(content, "quizNum") {
		//请求题目和发送答案的时候停止点按
		cancelTap()
		values, _ := url.ParseQuery(content)
		roomId, _ := strconv.Atoi(values.Get("roomID"))
		cacheKey := fmt.Sprintf("roomID=%s", strconv.Itoa(roomId))
		ctx.UserData = cacheKey
	} else if strings.Contains(content, "quiz") && strings.Contains(content, "options") {
		//获取到题目，将返回结果中注入答案,开始点击
		data = injectQuestionResponse(bytes, ctx)
	} else if strings.Contains(content, "score") && strings.Contains(content, "totalScore") {
		//答题结束，协程处理
		go cacheChooseResponse(bytes)
	}

	return
}

//收到结果
func cacheChooseResponse(bytes []byte) {

	var resp ChooseResp

	json.Unmarshal(bytes, &resp)

	cacheKey := fmt.Sprintf("roomID=%s", strconv.Itoa(resp.Data.RoomID))

	question := questions[cacheKey]

	if question.Quiz != "" {
		question.Answer = question.Options[resp.Data.Answer-1]
		cache(question)
		delete(questions, cacheKey)
	}

	if resp.Data.Num == 5 {
		log.Println("答题完毕！！！")
		gameRestart(12 * time.Second)
	}

	return
}

//收到题目开始点
func injectQuestionResponse(bytes []byte, ctx *goproxy.ProxyCtx) (data []byte) {
	var resp QuestionResp
	var origin QuestionResp

	json.Unmarshal(bytes, &resp)
	json.Unmarshal(bytes, &origin)

	cacheKey := ctx.UserData.(string)

	questions[cacheKey] = NewQuestion(origin)

	start := time.Now()

	answer := fetchAnswerFromCache(resp.Data.Quiz)

	//收到题目开始点答案

	guss := 0

	if answer != "" {
		for index, option := range resp.Data.Options {
			if option == answer {
				resp.Data.Options[index] = option + "[标答]"
				guss = index
			}
		}
	} else {

		page := search(resp.Data.Quiz)

		//如果题干中包含 否定字眼 结果取反
		var max, min, reverse = 0, 65535,
			strings.Contains(resp.Data.Quiz, "不是") ||
				strings.Contains(resp.Data.Quiz, "不属于") ||
				strings.Contains(resp.Data.Quiz, "不包括")

		for index, option := range resp.Data.Options {
			words := util.Split(option)

			grade := strings.Count(page, option)

			//log.Println(option + "加了" + strconv.Itoa(grade) + "权重")
			if len(words) > 1 {
				for _, word := range words {
					if len(word) > 1 {
						//分词的权重计算
						g := int(float32(strings.Count(page, option)) * (1 / float32(len(words))))
						grade += g
						//log.Println(word + "加了" + strconv.Itoa(g) + "权重")
					}
				}
			}

			resp.Data.Options[index] = option + "[" + strconv.Itoa(grade) + "]"

			if reverse {
				if grade < min {
					min = grade
					guss = index
				}
			} else {
				if grade > max {
					max = grade
					guss = index
				}
			}

		}

	}

	delta := time.Now().Sub(start)

	log.Printf("查找答案耗时: %s\n", delta)

	//延时点按
	tap(guss, (3200*time.Millisecond)-delta)

	log.Println(resp.Data.Quiz)

	for _, item := range resp.Data.Options {
		log.Println(item)
	}

	data, _ = json.Marshal(resp)

	return
}



//循环点按直到返回结果，不同分辨率按钮位置不同, 需要延时触发
func tap(i int, delay time.Duration) {

	go func() {
		//先取消点按
		cancelTap()
		//取消重启
		cancelRestart()
		//等待动画延时
		time.Sleep(delay - 1*time.Second)

		tapTask = time.NewTicker(1 * time.Second)
		for _ = range tapTask.C {
			switch i {
			case 0:
				util.RunWithAdb("shell", "input tap 540 1040")
				break
			case 1:
				util.RunWithAdb("shell", "input tap 540 1240")
				break
			case 2:
				util.RunWithAdb("shell", "input tap 540 1440")
				break
			case 3:
				util.RunWithAdb("shell", "input tap 540 1640")
				break
			}
		}

		//选择答案12秒后 没有再次触发点击事件则 重启游戏
		gameRestart(12 * time.Second)
	}()
}

//停止选择
func cancelTap()  {
	if tapTask != nil {
		tapTask.Stop()
	}
}

//取消游戏重启
func cancelRestart() {
	if restartTask != nil {
		 restartTask.Stop()
	}
}

//答题完毕后点击 继续游戏 ，但是这里可能会遇到弹出升级框的情况，有待优化
func gameRestart(delay time.Duration) {
	if restartTask != nil {
		restartTask.Stop()
	}

	restartTask = time.AfterFunc(delay, func() {
		cancelTap()
		//继续游戏 -> 重开
		util.RunWithAdb("shell", "input tap 540 1440")
		util.RunWithAdb("shell", "input tap 540 1740")
		//util.RunWithAdb("shell", "input tap 540 1240")
		//util.RunWithAdb("shell", "input tap 540 1690")
		//util.RunWithAdb("shell", "input tap 540 1240")
		//util.RunWithAdb("shell", "input tap 540 1690")
	})
}

func search(question string) string {
	req, _ := http.NewRequest("GET", "http://www.baidu.com/s?wd="+url.QueryEscape(question), nil)
	resp, _ := http.DefaultClient.Do(req)
	content, _ := ioutil.ReadAll(resp.Body)
	return string(content)
}

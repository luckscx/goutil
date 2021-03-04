package part

import (
    "sync"
    "io"
    "os"
    "context"
    "time"
    "strings"
    "net/http"
    "errors"
    "io/ioutil"
    "net/url"
    compress "github.com/qydysky/part/compress"
    // "encoding/binary"
)

type Rval struct {
    Url string
    PostStr string
    Timeout int
    Proxy string
    Retry int
    SleepTime int
    JustResponseCode bool
    SaveToPath string

    Header map[string]string
}

type req struct {
    Respon []byte
    Response  *http.Response
    UsedTime time.Duration

    cancelOpen bool
    cancel chan interface{}
    sync.Mutex
}

func Req() *req{
    return &req{}
}

// func main(){
//     var _ReqfVal = ReqfVal{
//         Url:url,
//         Proxy:proxy,
// 		Timeout:10,
// 		Retry:2,
//     }
//     Reqf(_ReqfVal)
// }

func (this *req) Reqf(val Rval) (error) {
    this.Lock()
	defer this.Unlock()

	var returnErr error

	_val := val;

    if _val.Timeout==0{_val.Timeout=3}

	for ;_val.Retry>=0;_val.Retry-- {
        returnErr=this.Reqf_1(_val)
        select {
        case <- this.cancel://cancel
            return returnErr
        default:
            if returnErr==nil {return nil}
        }
        time.Sleep(time.Duration(_val.SleepTime)*time.Millisecond)
    }

	return returnErr
}

func (this *req) Reqf_1(val Rval) (error) {

	var (
        Url string = val.Url
        PostStr string = val.PostStr
        Proxy string = val.Proxy
        Timeout int = val.Timeout
        JustResponseCode bool =val.JustResponseCode
        SaveToPath string =val.SaveToPath

        Header map[string]string = val.Header
    )

    var beginTime time.Time = time.Now()

    var client http.Client

    if Header == nil {Header = make(map[string]string)}

    if Timeout != -1 {
        client.Timeout = time.Duration(Timeout)*time.Second
    }

    if Proxy!="" {
        proxy := func(_ *http.Request) (*url.URL, error) {
            return url.Parse(Proxy)
        }
        client.Transport = &http.Transport{Proxy: proxy}
    } else {
        client.Transport = &http.Transport{}
    }
    
    if Url==""{return errors.New("Url is \"\"")}

    Method := "GET"
    var body io.Reader
    if len(PostStr) > 0 {
        Method = "POST";
        body = strings.NewReader(PostStr);
        if _,ok := Header["ContentType"];!ok {Header["ContentType"] = "application/x-www-form-urlencoded"}
    }

    cx, cancel := context.WithCancel(context.Background())
    req,_ := http.NewRequest(Method, Url, body)
    req = req.WithContext(cx)

    var done = make(chan struct{})
    defer close(done)
    go func(){
        this.cancel = make(chan interface{})
        this.cancelOpen = true
        select {
        case <- this.cancel:cancel()
        case <- done:
        }
    }()
    
    if _,ok := Header["Accept"];!ok {Header["Accept"] = `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8`}
    if _,ok := Header["Connection"];!ok {Header["Connection"] = "keep-alive"}
    if _,ok := Header["Accept-Encoding"];!ok {Header["Accept-Encoding"] = "gzip, deflate, br"}
    if SaveToPath != "" {Header["Accept-Encoding"] = "identity"}
    if _,ok := Header["User-Agent"];!ok {Header["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36"}

    for k,v := range Header {
        req.Header.Set(k, v)
    }

    resp, err := client.Do(req)

    if err != nil {
        return err
    }
    
    var saveToFile func(io.Reader,string)error = func (Body io.Reader,filepath string) error {
        out, err := os.Create(filepath + ".dtmp")
        if err != nil {out.Close();return err}

        // resp, err := http.Get(url)
        // if err != nil {out.Close();return err}
        // defer resp.Body.Close()

        if _, err = io.Copy(out, Body); err != nil {out.Close();return err}
        out.Close()

        if err = os.RemoveAll(filepath); err != nil {return err}
        if err = os.Rename(filepath+".dtmp", filepath); err != nil {return err}
        return nil
    }
    this.Response = resp
    if !JustResponseCode {
        defer resp.Body.Close()
        if SaveToPath != "" && resp.StatusCode == 200 {
            if err := saveToFile(resp.Body, SaveToPath); err != nil {
                return err
            }
        }else{
            var err error
            this.Respon,err = ioutil.ReadAll(resp.Body)
            if err != nil {return err}
            if compress_type := resp.Header[`Content-Encoding`];compress_type!=nil{
                switch compress_type[0]{
                case `br`:
                    if tmp,err := compress.UnBr(this.Respon);err != nil {
                        return err
                    }else{this.Respon = append([]byte{},tmp...)}
                case `gzip`:
                    if tmp,err := compress.UnGzip(this.Respon);err != nil {
                        return err
                    }else{this.Respon = append([]byte{},tmp...)}
                case `deflate`:
                    if tmp,err := compress.UnFlate(this.Respon);err != nil {
                        return err
                    }else{this.Respon = append([]byte{},tmp...)}
                default:
                }
            }
        }
    } else {resp.Body.Close()}
    
    this.UsedTime=time.Since(beginTime)
    
    return nil
}

func (t *req) Close(){
    if !t.cancelOpen {return}
    select {
    case <- t.cancel://had close
        return
    default:
        close(t.cancel)
        t.cancelOpen = false
    }
}

func Cookies_String_2_Map(Cookies string) (o map[string]string) {
    o = make(map[string]string)
    list := strings.Split(Cookies, `; `)
    for _,v := range list {
        s := strings.SplitN(v, "=", 2)
        if len(s) != 2 {continue}
        o[s[0]] = s[1]
    }
    return
}

func Map_2_Cookies_String(Cookies map[string]string) (o string) {
    if len(Cookies) == 0 {return ""}
    for k,v := range Cookies {
        o += k +`=`+ v + `; `
    }
    t := []rune(o)
    o = string(t[:len(t)-2])
    return
}

func Cookies_List_2_Map(Cookies []*http.Cookie) (o map[string]string) {
    o = make(map[string]string)
    for _,v := range Cookies {
		o[v.Name] = v.Value
    }
    return
}
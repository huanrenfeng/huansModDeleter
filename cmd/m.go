// web-page based mod deleter to remove mods according to the content of a 7z file
package main
import(
	"github.com/renfenghuan/huansModDeleter/g7z"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"golang.org/x/net/websocket"
	
	"path/filepath"
	"runtime"
	"os"
	"os/exec"
	"os/user"
	"os/signal"
	"archive/zip"
	"log"
	"bufio"
)

func ChangeToExeDir() {
	exe, _ := os.Executable()
	exepath := filepath.Dir(exe)
	os.Chdir(exepath)
}


func OpenBrowser(url string){
	var err interface{}

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = "unsupported platform"
	}
	if err != nil {
		log.Fatal(err)
	}

}

type PageData struct{
	The7zDirectory string
	TheGameDirectory string	
	Port string
	
}

var settingSaveFolderDir string
var data PageData

func settingFilePath() string{
	return settingSaveFolderDir+"\\setting.txt"
}

func readSetting(){

	file, err := os.Open(settingFilePath())
	defer file.Close()
	
	if err == nil{
		scanner := bufio.NewScanner(file)
		
		line:=0
		
		for scanner.Scan() {
			t:= scanner.Text()
			
			if t==""{
				line++
				continue
			}
				
			
			switch line{
			case 0:
				data.The7zDirectory = t
			case 1:
				data.TheGameDirectory = t
			case 2:
				data.Port = t
			default:
				break
			}
			
			line++
		}
		
		if line >= 2{
			return
		}
		
	}
	
	g7z.Detect7z()
	
	port:= "60002"
	
	data = PageData{
		g7z.The7zPath,
		"",
		port,
	}
}

func writeSetting(){
	fmt.Println("writeSetting")
	
	f, err := os.OpenFile(settingFilePath(), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		fmt.Println(err)
	}
	
	fmt.Fprintf(f,"%s\n",data.The7zDirectory)
	fmt.Fprintf(f,"%s\n",data.TheGameDirectory)
	fmt.Fprintf(f,"%s\n",data.Port)
	
	if err := f.Close(); err != nil {
		fmt.Println(err)
	}
}

func main(){
	
	
	
	//----------------------------------------------------------------------
	////------	Signal ----------
	
	killSignal := make(chan os.Signal, 1)
	signal.Notify(killSignal, os.Interrupt)
	
	go func(){
		<-killSignal
		//writeSetting()
		os.Exit(0)
	}()
	
	
	
	//----------------------------------------------------------------------
	//------	Setting ----------
	
	
	u, _ := user.Current()	
	settingSaveFolderDir = u.HomeDir+`\Documents\Huans Mod Deleter`
	os.MkdirAll(settingSaveFolderDir, os.ModePerm)

	readSetting()

	
	//----------------------------------------------------------------------
	//------	WebPage ----------
	
	
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	
       tmpl, err := template.ParseFiles("page.html")
	   
		if err!= nil{
			fmt.Println("page.html error")
			return
		}
		
        tmpl.Execute(w, data)
    })
	
	http.Handle("/ws", websocket.Handler( func(ws *websocket.Conn) {
       
		
        fmt.Println("new client connection")
		
		for {
			var reply string

			if err := websocket.Message.Receive(ws, &reply); err != nil {
				fmt.Println(err)
				return
			}
			
			fmt.Println(reply)
	 
			strs:=strings.SplitN(reply,":",2)
			
			if len(strs)>1{
				t:=strs[1]
				
				switch strs[0]{
				case "mod":
					
					tryDelete(strs[1],ws)
				case "cgd":
					t = strings.Title(t)
					
					if f,e:=os.Open(t);e== nil {
						if fi,e:=f.Stat();e==nil && fi.IsDir(){
							
							fmt.Println("Directory format correct")
							
							if filepath.Base(t) != "Data"{
								t = filepath.Join(t,"Data")
							}
							
							data.TheGameDirectory = t
							writeSetting()
						}
					}
					
				case "7f":
					data.The7zDirectory = t
					writeSetting()
				}
			}
			
		}
    }))
	
    go func(){
		OpenBrowser("http://localhost:"+data.Port)
	}()
	
	http.ListenAndServe(":"+data.Port, nil)
	

}

func tryDelete(fn string,ws *websocket.Conn){
	d:= data.TheGameDirectory
	if d == ""{
		return
	}

	if strings.HasSuffix(fn,`\`){
		fn = strings.TrimSuffix(fn,`\`)
	}

	fmt.Println("try to delete mod "+ fn)
	
	fn = strings.ToLower(fn)
	
	if strings.HasSuffix(fn,`.7z`){
	
		archive, err := g7z.NewArchive(fn)
		if err!= nil{
			fmt.Println(err)
			return
		}

		os.Chdir(d)
		
		// list all files inside archive
		for _, en := range archive.Entries {
			if en.IsDirectory(){
				continue
			}
			
			tfn:=en.Path
			
			tryDeleteFile(tfn)
		}
		
	}else if strings.HasSuffix(fn,`.zip`){
		r,err:= zip.OpenReader(fn)
		if err!=nil{
			fmt.Println(err)
			return
		}
		
		for _, f := range r.File {
			
			if f.FileHeader.FileInfo().IsDir(){
				continue
			}
			
			tfn:=f.Name
			
			tryDeleteFile(tfn)

		}
	}
}

func tryDeleteFile(fn string){
	d:= data.TheGameDirectory
	
	fn = strings.ToLower(fn)
	
	tfn:= strings.Title(fn)
			
	if strings.HasPrefix(tfn,`Data/`){
		tfn = strings.TrimPrefix(tfn,`Data/`)
		
	}else if strings.HasPrefix(tfn,`Data\`){
		tfn = strings.TrimPrefix(tfn,`Data\`)
	}
	
	fmt.Println("try delete ", tfn )
	
	if err:= os.Remove(filepath.Join(d,tfn));err==nil{
		fmt.Println("delete successfully")
	}else{
		fmt.Println("delete error:",err)
	}
}
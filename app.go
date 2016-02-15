package main

import (
    	"html/template"
	"io"
	"net/http"
	"os"
	"io/ioutil"
	"os/exec"
	"bytes"
	//"log"
	"fmt"
)

//Compile templates on start
var templates = template.Must(template.ParseFiles("tmpl/upload.html"))

func Pipeline(cmds ...*exec.Cmd) (pipeLineOutput, collectedStandardError []byte, pipeLineError error) {
        // Require at least one command
        if len(cmds) < 1 { 
                return nil, nil, nil
        }

        // Collect the output from the command(s)
        var output bytes.Buffer
        var stderr bytes.Buffer

        last := len(cmds) - 1
        for i, cmd := range cmds[:last] {
                var err error
                // Connect each command's stdin to the previous command's stdout
                if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
                        return nil, nil, err
                }
                // Connect each command's stderr to a buffer
                cmd.Stderr = &stderr
        }

        // Connect the output and error for the last command
        cmds[last].Stdout, cmds[last].Stderr = &output, &stderr

        // Start each command
        for _, cmd := range cmds {
                if err := cmd.Start(); err != nil {
                        return output.Bytes(), stderr.Bytes(), err
                }
        }

        // Wait for each command to complete
        for _, cmd := range cmds {
                if err := cmd.Wait(); err != nil {
                        return output.Bytes(), stderr.Bytes(), err
                }
        }

        // Return the pipeline output and the collected standard error
        return output.Bytes(), stderr.Bytes(), nil
}

//Display the named template
func display(w http.ResponseWriter, tmpl string, data interface{}) {
	templates.ExecuteTemplate(w, tmpl+".html", data)
}

func out_page(w http.ResponseWriter, tmpl string, data interface{}) {
	templates.ExecuteTemplate(w, tmpl+".html", data)
}

//This is where the action happens.
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	//GET displays the upload form.
	case "GET":
		display(w, "upload", nil)

	//POST takes the uploaded file(s) and saves it to disk.
	case "POST":
		//get the multipart reader for the request.
		reader, err := r.MultipartReader()
		//var z [][]byte
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var files []string
		var names []string
		var dirs []string
		//copy each part to destination.
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}

			//if part.FileName() is empty, skip this iteration.
			if part.FileName() == "" {
				continue
			}
			names = append(names, part.FileName())
			tmp_dir, _ := ioutil.TempDir(os.TempDir(), part.FileName())
			dirs = append(dirs, tmp_dir)
			tmp_file := fmt.Sprintf("%s/%s", tmp_dir, part.FileName())
			dst, err :=  os.Create(tmp_file)
			files = append(files, tmp_file)
			
			defer dst.Close()
			
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			
			if _, err := io.Copy(dst, part); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		}
		//display success message.
		//var t string
		var final string
		cmd := "pngquant/pngquant"
		for i,z := range files {
			//u := fmt.Sprintf("--ext=.png.x")
			if err := exec.Command(cmd, "--quality=60-90", z, "--ext=.png.x").Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			final += fmt.Sprintf("%s/%s.x", dirs[i], names[i])
			//t += fmt.Sprintf("%s ", z)
			//fmt.Fprintf(w, 
		}
		//display(w, "upload", final)
		fmt.Fprintf(w, final)		
		//os.RemoveAll(tmp_dir)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/upload", uploadHandler)

	//static file handler.
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	//Listen on port 8080
	http.ListenAndServe(":8080", nil)
}

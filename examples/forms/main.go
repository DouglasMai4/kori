package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/douglasmai4/kori"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ContactInput struct {
	Name      string `form:"name" validate:"required,min=2,max=100"`
	Email     string `form:"email" validate:"required,email"`
	Message   string `form:"message" validate:"required,min=10,max=1000"`
	Subscribe bool   `form:"subscribe"`
}

type ProfileInput struct {
	DisplayName string                `form:"display_name" validate:"required,min=2,max=50"`
	Bio         string                `form:"bio" validate:"max=500"`
	Avatar      *multipart.FileHeader `form:"avatar" validate:"required"`
}

type GalleryInput struct {
	Title  string                    `form:"title" validate:"required"`
	Tags   []string                  `form:"tags" validate:"required,min=1,dive,min=2"`
	Photos []*multipart.FileHeader   `form:"photos" validate:"required,min=1"`
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/", serveHomePage)

	kori.POST(r, "/contact", handleContact)
	kori.POST(r, "/profile", handleProfile)
	kori.POST(r, "/gallery", handleGallery)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("forms demo listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func serveHomePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<title>Kori Forms Demo</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: sans-serif; margin: 2rem; max-width: 800px; }
  h2 { border-bottom: 2px solid #eee; padding-bottom: .5rem; margin-top: 2rem; }
  form { background: #f9f9f9; padding: 1.5rem; border-radius: 8px; margin: 1rem 0; }
  label { display: block; margin: .75rem 0 .25rem; font-weight: 600; }
  input, textarea, select { width: 100%; padding: .5rem; border: 1px solid #ccc; border-radius: 4px; font: inherit; }
  textarea { resize: vertical; min-height: 80px; }
  input[type=checkbox] { width: auto; margin-right: .5rem; }
  input[type=file] { padding: .25rem 0; }
  button { background: #4f46e5; color: #fff; border: none; padding: .75rem 1.5rem; border-radius: 4px; font: inherit; cursor: pointer; margin-top: 1rem; }
  button:hover { background: #4338ca; }
  .hint { font-size: .85rem; color: #666; margin-top: .25rem; }
  .tag-group { display: flex; gap: 1rem; flex-wrap: wrap; }
  .tag-group label { font-weight: 400; display: inline-flex; align-items: center; gap: .25rem; }
  .result { background: #ecfdf5; border: 1px solid #a7f3d0; padding: 1rem; border-radius: 4px; margin-top: 1rem; white-space: pre-wrap; font-family: monospace; }
</style>
</head>
<body>
<h1>Kori Forms Demo</h1>
<p>Demonstrates <code>BindForm</code> (URL-encoded) and <code>BindMultipart</code> (file uploads).</p>

<h2>Contact Form — <code>BindForm</code></h2>
<form action="/contact" method="post" id="contactForm">
  <label for="name">Name</label>
  <input type="text" name="name" id="name" required>

  <label for="email">Email</label>
  <input type="email" name="email" id="email" required>

  <label for="message">Message</label>
  <textarea name="message" id="message" required></textarea>

  <label>
    <input type="checkbox" name="subscribe" value="true">
    Subscribe to newsletter
  </label>

  <button type="submit">Submit Contact</button>
</form>
<div id="contactResult" class="result"></div>

<h2>Profile Upload — <code>BindMultipart</code></h2>
<form action="/profile" method="post" enctype="multipart/form-data" id="profileForm">
  <label for="displayName">Display Name</label>
  <input type="text" name="display_name" id="displayName" required>

  <label for="bio">Bio</label>
  <textarea name="bio" id="bio" placeholder="Tell us about yourself..."></textarea>

  <label for="avatar">Avatar (required)</label>
  <input type="file" name="avatar" id="avatar" accept="image/*" required>
  <div class="hint">Single file upload using <code>*multipart.FileHeader</code></div>

  <button type="submit">Upload Profile</button>
</form>
<div id="profileResult" class="result"></div>

<h2>Gallery Upload — <code>BindMultipart</code> with slices</h2>
<form action="/gallery" method="post" enctype="multipart/form-data" id="galleryForm">
  <label for="title">Gallery Title</label>
  <input type="text" name="title" id="title" required>

  <label>Tags</label>
  <div class="tag-group">
    <label><input type="checkbox" name="tags" value="go"> Go</label>
    <label><input type="checkbox" name="tags" value="web"> Web</label>
    <label><input type="checkbox" name="tags" value="api"> API</label>
    <label><input type="checkbox" name="tags" value="demo"> Demo</label>
  </div>
  <div class="hint">Slice field: multiple values with the same <code>tags</code> name</div>

  <label for="photos">Photos (multiple)</label>
  <input type="file" name="photos" id="photos" multiple accept="image/*">
  <div class="hint">Multiple file upload using <code>[]*multipart.FileHeader</code></div>

  <button type="submit">Upload Gallery</button>
</form>
<div id="galleryResult" class="result"></div>

<script>
async function submitForm(formId, resultId) {
  const form = document.getElementById(formId);
  const result = document.getElementById(resultId);
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    result.textContent = "Submitting...";
    try {
      const res = await fetch(form.action, { method: "POST", body: new FormData(form) });
      const data = await res.json();
      result.textContent = JSON.stringify(data, null, 2);
      result.style.borderColor = res.ok ? "#a7f3d0" : "#fca5a5";
      result.style.background = res.ok ? "#ecfdf5" : "#fef2f2";
    } catch (err) {
      result.textContent = "Error: " + err.message;
      result.style.borderColor = "#fca5a5";
      result.style.background = "#fef2f2";
    }
  });
}
submitForm("contactForm", "contactResult");
submitForm("profileForm", "profileResult");
submitForm("galleryForm", "galleryResult");
</script>
</body>
</html>`)
}

func handleContact(w http.ResponseWriter, r *http.Request) error {
	var input ContactInput
	if err := kori.BindForm(r, &input); err != nil {
		return err
	}
	return kori.JSON(w, http.StatusOK, map[string]any{
		"message":   "contact submitted successfully",
		"name":      input.Name,
		"email":     input.Email,
		"subscribe": input.Subscribe,
	})
}

func handleProfile(w http.ResponseWriter, r *http.Request) error {
	var input ProfileInput
	if err := kori.BindMultipart(r, &input); err != nil {
		return err
	}
	saved, err := saveFile(input.Avatar, "avatars")
	if err != nil {
		return kori.InternalServerError("failed to save avatar", err.Error())
	}
	return kori.JSON(w, http.StatusOK, map[string]any{
		"message":      "profile updated",
		"display_name": input.DisplayName,
		"bio":          input.Bio,
		"avatar":       saved,
	})
}

func handleGallery(w http.ResponseWriter, r *http.Request) error {
	var input GalleryInput
	if err := kori.BindMultipart(r, &input); err != nil {
		return err
	}
	var files []string
	for _, fh := range input.Photos {
		saved, err := saveFile(fh, "gallery")
		if err != nil {
			return kori.InternalServerError("failed to save photo", err.Error())
		}
		files = append(files, saved)
	}
	return kori.JSON(w, http.StatusOK, map[string]any{
		"message": "gallery created",
		"title":   input.Title,
		"tags":    input.Tags,
		"photos":  files,
	})
}

func saveFile(fh *multipart.FileHeader, subdir string) (string, error) {
	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dir := filepath.Join("uploads", subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fh.Filename)
	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return filepath.Join(subdir, name), nil
}

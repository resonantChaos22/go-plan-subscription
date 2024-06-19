package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
	"github.com/resonantchaos22/go-concur-final/data"
)

const (
	APP_URL = "http://localhost:3000"
)

func (app *Config) HomePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "home.page.gohtml", nil)
}

func (app *Config) LoginPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "login.page.gohtml", nil)
}

func (app *Config) PostLoginPage(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}

	//	get email and password from form post
	email := r.Form.Get("email")
	password := r.Form.Get("password")

	user, err := app.Models.User.GetByEmail(email)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Invalid Credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		app.ErrorLog.Println(err)
		return
	}

	if user.Active != 1 {
		app.Session.Put(r.Context(), "error", "Invalid Credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		app.ErrorLog.Println("User is not activated yet")
		return
	}

	//	check password
	validPassword, err := user.PasswordMatches(password)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Invalid Credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		app.ErrorLog.Println(err)
		return
	}
	if !validPassword {
		msg := Message{
			To:      email,
			Subject: "Failed Login Attempt",
			Data:    "Invalid Login Attempt",
		}
		app.sendEmail(msg)

		app.Session.Put(r.Context(), "error", "Invalid Credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//	user log in
	app.Session.Put(r.Context(), "userID", user.ID)
	app.Session.Put(r.Context(), "user", user)

	app.Session.Put(r.Context(), "flash", "Successful login!")

	//	redirect the user
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Config) Logout(w http.ResponseWriter, r *http.Request) {
	app.Session.Pop(r.Context(), "userID")
	app.Session.Pop(r.Context(), "user")
	app.Session.Put(r.Context(), "flash", "Logged Out Successfully")

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *Config) RegisterPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "register.page.gohtml", nil)
}

func (app *Config) PostRegisterPage(w http.ResponseWriter, r *http.Request) {
	//*	create a user
	_ = app.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}

	//TODO	validate data

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	firstName := r.Form.Get("first-name")
	lastName := r.Form.Get("last-name")

	createUserReq := data.User{
		FirstName: firstName,
		LastName:  lastName,
		Password:  password,
		Email:     email,
		Active:    0,
		IsAdmin:   0,
	}
	userID, err := app.Models.User.Insert(createUserReq)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to create User.")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		app.ErrorLog.Println(err)
		return
	}

	user, err := app.Models.User.GetOne(userID)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Something went wrong!")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		app.ErrorLog.Println(err)
		return
	}

	//*	send an activation mail
	url := fmt.Sprintf("%s/activate?email=%s", APP_URL, user.Email)
	signedURL := GenerateTokenFromString(url)

	msg := Message{
		To:       user.Email,
		Subject:  "Activate your account",
		Template: "confirmation-email",
		Data:     template.HTML(signedURL),
	}
	app.sendEmail(msg)

	app.Session.Put(r.Context(), "flash", "Confirmation email sent. Check your email.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)

	// //	user log in
	// app.Session.Put(r.Context(), "userID", user.ID)
	// app.Session.Put(r.Context(), "user", user)

	// app.Session.Put(r.Context(), "flash", "Successful Sign Up!")

	// //	redirect the user
	// http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Config) ActivateAccount(w http.ResponseWriter, r *http.Request) {
	//TODO	validate URL
	url := r.RequestURI
	log.Println(url)
	testURL := fmt.Sprintf("http://localhost:3000%s", url)
	okay := VerifyToken(testURL)

	if !okay {
		app.Session.Put(r.Context(), "error", "Invalid Token.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	//	activate account
	u, err := app.Models.User.GetByEmail(r.URL.Query().Get("email"))
	if err != nil {
		app.Session.Put(r.Context(), "error", "No user found.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	u.Active = 1
	err = u.Update()
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to Update User.")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "flash", "Account activated. You can now log in.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)

}

func (app *Config) SubscribeToPlan(w http.ResponseWriter, r *http.Request) {
	// get the id of the chosen plan
	id := r.URL.Query().Get("id")
	planID, _ := strconv.Atoi(id)

	// get the plan from the database
	plan, err := app.Models.Plan.GetOne(planID)
	if err != nil {
		app.Session.Put(r.Context(), "error", "No plan found.")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}
	// get the user from session
	user, ok := app.Session.Get(r.Context(), "user").(data.User)
	if !ok {
		app.Session.Put(r.Context(), "error", "Log in first!")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//TODO	generate an invoice and email it
	app.WaitGroup.Add(1)
	go func() {
		defer app.WaitGroup.Done()
		invoice, err := app.getInvoice(user, plan)
		if err != nil {
			app.ErrorChan <- err
		}

		msg := Message{
			To:       user.Email,
			Subject:  "Your invoice!",
			Data:     invoice,
			Template: "invoice",
		}
		app.sendEmail(msg)
	}()

	//TODO	generate manual
	app.WaitGroup.Add(1)
	go func() {
		defer app.WaitGroup.Done()

		pdf := app.generateManual(user, plan)
		fileName := fmt.Sprintf("%d_%d_manual.pdf", user.ID, time.Now().Unix())
		err := pdf.OutputFileAndClose(fmt.Sprintf("./tmp/%s", fileName))
		if err != nil {
			app.ErrorChan <- err
			return
		}

		msg := Message{
			To:      user.Email,
			Subject: "Your Manual",
			Data:    "Your user manual is attached",
			AttachmentMap: map[string]string{
				"Manual.pdf": fmt.Sprintf("./tmp/%s", fileName),
			},
		}

		app.sendEmail(msg)

		//	test error chan
		app.ErrorChan <- fmt.Errorf("some error")
	}()

	//TODO	subscribe the user to an account
	err = app.Models.Plan.SubscribeUserToPlan(user, *plan)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to subscribe to plan.")
		app.ErrorLog.Println(err)
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}
	user.Plan = plan
	app.Session.Put(r.Context(), "user", user)

	//	Redirect
	app.Session.Put(r.Context(), "flash", fmt.Sprintf("Subscribed to %s", plan.PlanName))
	http.Redirect(w, r, "/members/plans", http.StatusSeeOther)

}

func (app *Config) generateManual(u data.User, plan *data.Plan) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(10, 13, 10)

	importer := gofpdi.NewImporter()
	time.Sleep(5 * time.Second)

	t := importer.ImportPage(pdf, "./pdf/manual.pdf", 1, "/MediaBox")
	pdf.AddPage()

	importer.UseImportedTemplate(pdf, t, 0, 0, 215.9, 0)
	pdf.SetX(75)
	pdf.SetY(150)

	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(0, 4, fmt.Sprintf("%s %s", u.FirstName, u.LastName), "", "C", false)
	pdf.Ln(5)
	pdf.MultiCell(0, 4, fmt.Sprintf("%s User Guide", plan.PlanName), "", "C", false)

	return pdf

}

func (app *Config) getInvoice(u data.User, plan *data.Plan) (string, error) {
	return plan.PlanAmountFormatted, nil
}

func (app *Config) ChooseSubscription(w http.ResponseWriter, r *http.Request) {
	if !app.Session.Exists(r.Context(), "userID") {
		app.Session.Put(r.Context(), "warning", "You must log in to see the page")
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	plans, err := app.Models.Plan.GetAll()
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}

	dataMap := make(map[string]any)
	dataMap["plans"] = plans

	app.render(w, r, "plans.page.gohtml", &TemplateData{
		Data: dataMap,
	})
}

package mail

import (
	"fmt"
	"strings"
	"time"

	"haircutz/backend/internal/model"
)

type AppointmentPaidData struct {
	ShellData
	CustomerName   string
	CustomerEmail  string
	CustomerPhone  string
	Address        string
	TrackingNumber string
	HairstyleName  string
	ServiceLabel   string
	WhenLabel      string
	AmountPaid     string
	TrackPageURL   string
}

func serviceLabel(s model.ServiceType) string {
	if s == model.ServiceHomeService {
		return "Home service"
	}
	return "Walk-in"
}

func whenLabel(start, end time.Time) string {
	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		loc = time.UTC
	}
	s := start.In(loc)
	e := end.In(loc)
	return fmt.Sprintf("%s %s–%s", s.Format("Mon 2 Jan 2006"), s.Format("15:04"), e.Format("15:04"))
}

func AppointmentCustomerPaidEmail(a *model.Appointment, trackPageURL string) (subject, html, plain string, err error) {
	subject = "Haircutz by Larry — booking confirmed"
	shell := NewShellData(subject, "Booking confirmed")
	data := AppointmentPaidData{
		ShellData:      shell,
		CustomerName:   strings.TrimSpace(a.Customer.Name),
		TrackingNumber: a.TrackingNumber,
		HairstyleName:  a.Hairstyle.Name,
		ServiceLabel:   serviceLabel(a.ServiceType),
		WhenLabel:      whenLabel(a.StartAt, a.EndAt),
		AmountPaid:     FormatNGN(a.TotalAmountKobo),
		TrackPageURL:   trackPageURL,
	}
	html, err = Render("appointment-customer-paid", data)
	if err != nil {
		return "", "", "", err
	}
	plain = fmt.Sprintf(
		"%s\n\nYour Haircutz by Larry booking is confirmed.\n\nTracking: %s\nStyle: %s\nService: %s\nWhen: %s\nPaid: %s\n\nTrack: %s\n",
		greetingLine(a.Customer.Name),
		a.TrackingNumber,
		a.Hairstyle.Name,
		serviceLabel(a.ServiceType),
		whenLabel(a.StartAt, a.EndAt),
		FormatNGN(a.TotalAmountKobo),
		trackPageURL,
	)
	return subject, html, plain, nil
}

func AppointmentAdminPaidEmail(a *model.Appointment) (subject, html, plain string, err error) {
	subject = "Haircutz by Larry: new paid appointment"
	shell := NewShellData(subject, "New paid appointment")
	data := AppointmentPaidData{
		ShellData:      shell,
		CustomerName:   strings.TrimSpace(a.Customer.Name),
		CustomerEmail:  a.Customer.Email,
		CustomerPhone:  a.Customer.Phone,
		Address:        a.Customer.Address,
		TrackingNumber: a.TrackingNumber,
		HairstyleName:  a.Hairstyle.Name,
		ServiceLabel:   serviceLabel(a.ServiceType),
		WhenLabel:      whenLabel(a.StartAt, a.EndAt),
		AmountPaid:     FormatNGN(a.TotalAmountKobo),
	}
	html, err = Render("appointment-admin-paid", data)
	if err != nil {
		return "", "", "", err
	}
	plain = fmt.Sprintf(
		"New paid appointment\nTracking: %s\nCustomer: %s (%s)\nPhone: %s\nAddress: %s\nStyle: %s\nService: %s\nWhen: %s\nAmount: %s\n",
		a.TrackingNumber,
		a.Customer.Name,
		a.Customer.Email,
		a.Customer.Phone,
		a.Customer.Address,
		a.Hairstyle.Name,
		serviceLabel(a.ServiceType),
		whenLabel(a.StartAt, a.EndAt),
		FormatNGN(a.TotalAmountKobo),
	)
	return subject, html, plain, nil
}

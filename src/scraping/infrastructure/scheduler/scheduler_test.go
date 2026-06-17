package scheduler

import (
	"testing"
	"time"
)

// TestNextRunFromCron_ValidExpression verifica que un cron_expression estándar
// produce el próximo instante correcto a partir de una referencia conocida.
//
// Caso: "0 6 * * 1" (lunes a las 06:00).
// Referencia: miércoles 2026-06-10 12:00 UTC.
// Esperado: lunes 2026-06-15 06:00 UTC (el lunes siguiente).
func TestNextRunFromCron_ValidExpression(t *testing.T) {
	expr := "0 6 * * 1" // lunes 06:00

	// Miércoles 2026-06-10 12:00:00 UTC
	from := time.Date(2026, time.June, 10, 12, 0, 0, 0, time.UTC)

	next := nextRunFromCron(expr, from, nil)

	// El lunes siguiente es 2026-06-15 06:00:00 UTC
	expected := time.Date(2026, time.June, 15, 6, 0, 0, 0, time.UTC)

	if !next.Equal(expected) {
		t.Errorf("nextRunFromCron(%q, %v) = %v; want %v", expr, from, next, expected)
	}
}

// TestNextRunFromCron_FromJustBefore verifica que si la referencia es el mismo
// lunes justo antes de las 06:00, el resultado es ese mismo lunes a las 06:00.
func TestNextRunFromCron_FromJustBefore(t *testing.T) {
	expr := "0 6 * * 1" // lunes 06:00

	// Lunes 2026-06-15 05:59:59 UTC — todavía no llegó la ventana
	from := time.Date(2026, time.June, 15, 5, 59, 59, 0, time.UTC)

	next := nextRunFromCron(expr, from, nil)

	// Debe disparar ese mismo lunes a las 06:00
	expected := time.Date(2026, time.June, 15, 6, 0, 0, 0, time.UTC)

	if !next.Equal(expected) {
		t.Errorf("nextRunFromCron(%q, %v) = %v; want %v", expr, from, next, expected)
	}
}

// TestNextRunFromCron_InvalidExpression verifica que un cron_expression inválido
// cae al fallback (from + schedulingFallbackDuration) sin panic.
func TestNextRunFromCron_InvalidExpression(t *testing.T) {
	expr := "no-es-un-cron-valido"
	from := time.Now()

	next := nextRunFromCron(expr, from, nil)

	// El resultado debe estar dentro del rango del fallback.
	// Permitimos ±1 segundo de tolerancia por el tiempo de ejecución.
	expectedMin := from.Add(schedulingFallbackDuration - time.Second)
	expectedMax := from.Add(schedulingFallbackDuration + time.Second)

	if next.Before(expectedMin) || next.After(expectedMax) {
		t.Errorf("nextRunFromCron con expr inválido = %v; esperaba ~%v (fallback %s)",
			next, from.Add(schedulingFallbackDuration), schedulingFallbackDuration)
	}
}

// TestNextRunFromCron_EmptyExpression verifica que un cron_expression vacío
// aplica el fallback y no entra en loop.
func TestNextRunFromCron_EmptyExpression(t *testing.T) {
	from := time.Now()

	next := nextRunFromCron("", from, nil)

	expectedMin := from.Add(schedulingFallbackDuration - time.Second)
	expectedMax := from.Add(schedulingFallbackDuration + time.Second)

	if next.Before(expectedMin) || next.After(expectedMax) {
		t.Errorf("nextRunFromCron con expr vacío = %v; esperaba ~%v (fallback %s)",
			next, from.Add(schedulingFallbackDuration), schedulingFallbackDuration)
	}
}

// TestNextRunFromCron_NoLoop verifica la propiedad fundamental de corrección:
// el valor devuelto siempre es estrictamente posterior a `from`, lo que garantiza
// que la fuente no vuelva a estar "due" de forma inmediata.
func TestNextRunFromCron_NoLoop(t *testing.T) {
	cases := []struct {
		name string
		expr string
	}{
		{"cada minuto", "* * * * *"},
		{"lunes 06:00", "0 6 * * 1"},
		{"diario mediodía", "0 12 * * *"},
		{"inválido", "xyz"},
		{"vacío", ""},
	}

	from := time.Now()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next := nextRunFromCron(tc.expr, from, nil)
			if !next.After(from) {
				t.Errorf("nextRunFromCron(%q) devolvió %v que NO es posterior a from=%v — loop garantizado",
					tc.expr, next, from)
			}
		})
	}
}

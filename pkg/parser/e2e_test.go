package parser_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

func TestIsE2EFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		src      []byte
		want     bool
	}{
		// Path segment matches
		{
			name:     "path segment integration_test",
			filePath: "integration_test/app_test.dart",
			src:      []byte("// test"),
			want:     true,
		},
		{
			name:     "path segment integration_test nested with backslash",
			filePath: "foo\\integration_test\\bar_test.go",
			src:      nil,
			want:     true,
		},
		{
			name:     "path segment e2e",
			filePath: "test/e2e/login_test.go",
			src:      nil,
			want:     true,
		},
		{
			name:     "path segment playwright",
			filePath: "src/playwright/checkout.test.ts",
			src:      nil,
			want:     true,
		},
		{
			name:     "path segment cypress",
			filePath: "tests/cypress/spec.js",
			src:      nil,
			want:     true,
		},

		// File name ends with suffix matches
		{
			name:     "suffix .e2e.ts",
			filePath: "src/tests/auth.e2e.ts",
			src:      nil,
			want:     true,
		},
		{
			name:     "suffix .e2e.tsx",
			filePath: "src/tests/auth.e2e.tsx",
			src:      nil,
			want:     true,
		},
		{
			name:     "suffix .e2e.js",
			filePath: "src/tests/auth.e2e.js",
			src:      nil,
			want:     true,
		},
		{
			name:     "suffix .e2e-spec.ts",
			filePath: "test/auth.e2e-spec.ts",
			src:      nil,
			want:     true,
		},
		{
			name:     "suffix .cy.ts",
			filePath: "specs/navigation.cy.ts",
			src:      nil,
			want:     true,
		},
		{
			name:     "suffix .cy.js",
			filePath: "specs/navigation.cy.js",
			src:      nil,
			want:     true,
		},

		// TS/JS src imports @playwright/test or cypress
		{
			name:     "TS imports @playwright/test",
			filePath: "tests/checkout.test.ts",
			src:      []byte("import { test, expect } from '@playwright/test';"),
			want:     true,
		},
		{
			name:     "TS multiline import @playwright/test",
			filePath: "tests/checkout.test.ts",
			src: []byte(`import {
	test,
	expect
} from "@playwright/test";`),
			want: true,
		},
		{
			name:     "JS side-effect import @playwright/test",
			filePath: "tests/checkout.test.js",
			src:      []byte("import '@playwright/test';"),
			want:     true,
		},
		{
			name:     "TS imports cypress",
			filePath: "tests/login.test.ts",
			src:      []byte("import cypress from 'cypress';"),
			want:     true,
		},
		{
			name:     "JS require @playwright/test",
			filePath: "tests/checkout.test.js",
			src:      []byte("const { test } = require('@playwright/test');"),
			want:     true,
		},
		{
			name:     "JS require cypress",
			filePath: "tests/login.test.js",
			src:      []byte("const cypress = require('cypress');"),
			want:     true,
		},
		{
			name:     "TS dynamic import @playwright/test",
			filePath: "tests/dynamic.test.ts",
			src:      []byte("const pw = await import('@playwright/test');"),
			want:     true,
		},

		// Dart src contains IntegrationTestWidgetsFlutterBinding
		{
			name:     "Dart with IntegrationTestWidgetsFlutterBinding",
			filePath: "test/app_test.dart",
			src:      []byte("void main() { IntegrationTestWidgetsFlutterBinding.ensureInitialized(); }"),
			want:     true,
		},

		// Required false cases
		{
			name:     "Button.test.tsx with RTL imports",
			filePath: "src/components/Button.test.tsx",
			src: []byte(`import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Button from './Button';

test('renders button', () => {
	render(<Button label="Click" />);
	expect(screen.getByText('Click')).toBeInTheDocument();
});`),
			want: false,
		},
		{
			name:     "widget home_test.dart without binding",
			filePath: "test/widget/home_test.dart",
			src: []byte(`import 'package:flutter_test/flutter_test.dart';
void main() {
	testWidgets('renders home', (tester) async {
		await tester.pumpWidget(const MyApp());
		expect(find.text('Home'), findsOneWidget);
	});
}`),
			want: false,
		},
		{
			name:     "path containing integration_tests_helper not exact segment match",
			filePath: "src/integration_tests_helper/foo.test.ts",
			src:      []byte("import { helper } from './helper';"),
			want:     false,
		},

		// Additional false cases
		{
			name:     "standard Go unit test",
			filePath: "pkg/parser/brace_test.go",
			src:      []byte("package parser\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}"),
			want:     false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := parser.IsE2EFile(tc.filePath, tc.src)
			if got != tc.want {
				t.Errorf("IsE2EFile(%q, ...) = %v, want %v", tc.filePath, got, tc.want)
			}
		})
	}
}

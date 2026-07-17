import { page } from 'vitest/browser';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render } from 'vitest-browser-svelte';
import LoginPage from '../routes/(auth)/login/+page.svelte';
import TestWrapper from '@/test-utils/test-wrapper.svelte';
import { queryClient } from '@/query';

// ---------------------------------------------------------------------------
// Hoisted mock setup — these run before module resolution so every import
// sees the mocked versions.
// ---------------------------------------------------------------------------

const { mockPost, mockGoto, mockToastError } = vi.hoisted(() => ({
	mockPost: vi.fn(),
	mockGoto: vi.fn(),
	mockToastError: vi.fn()
}));

vi.mock('$app/navigation', () => ({
	goto: mockGoto
}));

vi.mock('$app/paths', () => ({
	resolve: (path: string) => path
}));

vi.mock('@/axios', () => ({
	api: {
		post: mockPost,
		get: vi.fn()
	},
	axiosErr: vi.fn()
}));

vi.mock('svelte-sonner', () => ({
	toast: {
		error: mockToastError,
		success: vi.fn()
	}
}));

// ---------------------------------------------------------------------------
// Shared fixture data
// ---------------------------------------------------------------------------

const validCredentials = {
	email: 'john@example.com',
	password: 'secure-password'
};

const mockAuthResponse = {
	message: 'Login successful',
	data: {
		name: 'John Doe',
		email: 'john@example.com',
		org_id: 'org-123',
		org_name: 'My Org'
	}
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('Login Page', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		queryClient.clear();
	});

	it('renders the login form with all fields', async () => {
		render(TestWrapper, { props: { component: LoginPage } });

		await expect
			.element(page.getByRole('heading', { level: 1 }))
			.toHaveTextContent('Log in to your account');
		await expect.element(page.getByPlaceholder('name@example.com')).toBeInTheDocument();
		await expect.element(page.getByLabelText('Password')).toBeInTheDocument();
		await expect.element(page.getByText('Remember me')).toBeInTheDocument();
		await expect.element(page.getByRole('button', { name: 'Log in' })).toBeInTheDocument();
	});

	it('shows a validation error when email is invalid', async () => {
		render(TestWrapper, { props: { component: LoginPage } });

		const emailInput = page.getByPlaceholder('name@example.com');
		await emailInput.fill('not-an-email');
		emailInput.element().blur();

		await expect.element(page.getByText('Please enter a valid email')).toBeInTheDocument();
	});

	it('shows a validation error when password is too short', async () => {
		render(TestWrapper, { props: { component: LoginPage } });

		const passwordInput = page.getByLabelText('Password');
		await passwordInput.fill('short');
		passwordInput.element().blur();

		await expect
			.element(page.getByText('Password must be at least 8 characters'))
			.toBeInTheDocument();
	});

	it('disables the submit button while the mutation is pending', async () => {
		// Keep the promise unresolved so the mutation stays in "pending" state
		mockPost.mockReturnValue(new Promise(() => {}));

		render(TestWrapper, { props: { component: LoginPage } });

		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByLabelText('Password');
		const submitButton = page.getByRole('button', { name: 'Log in' });

		await emailInput.fill(validCredentials.email);
		await passwordInput.fill(validCredentials.password);
		await submitButton.click();

		await expect.element(page.getByRole('button', { name: /logging in/i })).toBeInTheDocument();
	});

	it('calls the login API and navigates home on success', async () => {
		mockPost.mockResolvedValue({ data: mockAuthResponse });

		render(TestWrapper, { props: { component: LoginPage } });

		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByLabelText('Password');
		const submitButton = page.getByRole('button', { name: 'Log in' });

		await emailInput.fill(validCredentials.email);
		await passwordInput.fill(validCredentials.password);
		await submitButton.click();

		await vi.waitFor(() => {
			expect(mockPost).toHaveBeenCalledWith('/auth/login', {
				email: validCredentials.email,
				password: validCredentials.password
			});
		});

		await vi.waitFor(() => {
			expect(mockGoto).toHaveBeenCalledWith('/');
		});
	});

	it('shows an error toast on login failure', async () => {
		mockPost.mockRejectedValue(new Error('Invalid credentials'));

		render(TestWrapper, { props: { component: LoginPage } });

		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByLabelText('Password');
		const submitButton = page.getByRole('button', { name: 'Log in' });

		await emailInput.fill(validCredentials.email);
		await passwordInput.fill(validCredentials.password);
		await submitButton.click();

		await vi.waitFor(() => {
			expect(mockToastError).toHaveBeenCalledWith('Invalid credentials');
		});
	});

	it('provides a link to the register page', async () => {
		render(TestWrapper, { props: { component: LoginPage } });

		await expect.element(page.getByText("Don't have an account?")).toBeInTheDocument();
		const signUpLink = page.getByRole('link', { name: 'Sign up' });
		await expect.element(signUpLink).toBeInTheDocument();
	});
});

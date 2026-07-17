import { page } from 'vitest/browser';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render } from 'vitest-browser-svelte';
import RegisterPage from '../routes/(auth)/register/+page.svelte';
import TestWrapper from '@/test-utils/test-wrapper.svelte';
import { queryClient } from '@/query';

// ---------------------------------------------------------------------------
// Hoisted mock setup
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

const validPayload = {
	name: 'John Doe',
	email: 'john@example.com',
	password: 'secure-password',
	org_name: 'Acme Inc.'
};

const mockAuthResponse = {
	message: 'Registration successful',
	data: {
		name: 'John Doe',
		email: 'john@example.com',
		org_id: 'org-123',
		org_name: 'Acme Inc.'
	}
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('Register Page', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		queryClient.clear();
	});

	it('renders the registration form with all fields', async () => {
		render(TestWrapper, { props: { component: RegisterPage } });

		await expect
			.element(page.getByRole('heading', { level: 1 }))
			.toHaveTextContent('Create an account');
		await expect.element(page.getByLabelText('Name')).toBeInTheDocument();
		await expect.element(page.getByPlaceholder('name@example.com')).toBeInTheDocument();
		await expect.element(page.getByLabelText('Password')).toBeInTheDocument();
		await expect.element(page.getByLabelText('organization')).toBeInTheDocument();
		await expect.element(page.getByText('Remember me')).toBeInTheDocument();
		await expect.element(page.getByRole('button', { name: 'Create account' })).toBeInTheDocument();
	});

	it('shows validation error when name is too short', async () => {
		render(TestWrapper, { props: { component: RegisterPage } });

		const nameInput = page.getByPlaceholder('John Doe');
		await nameInput.fill('ab');
		nameInput.element().blur();

		await expect.element(page.getByText('Name must be at least 3 characters')).toBeInTheDocument();
	});

	it('shows validation error when email is invalid', async () => {
		render(TestWrapper, { props: { component: RegisterPage } });

		const emailInput = page.getByPlaceholder('name@example.com');
		await emailInput.fill('not-an-email');
		emailInput.element().blur();

		await expect.element(page.getByText('Please enter a valid email')).toBeInTheDocument();
	});

	it('shows validation error when password is too short', async () => {
		render(TestWrapper, { props: { component: RegisterPage } });

		const passwordInput = page.getByLabelText('Password');
		await passwordInput.fill('short');
		passwordInput.element().blur();

		await expect
			.element(page.getByText('Password must be at least 8 characters'))
			.toBeInTheDocument();
	});

	it('shows validation error when organization is too short', async () => {
		render(TestWrapper, { props: { component: RegisterPage } });

		const orgInput = page.getByPlaceholder('Acme Inc.');
		await orgInput.fill('ab');
		orgInput.element().blur();

		await expect
			.element(page.getByText('organization must be at least 3 characters'))
			.toBeInTheDocument();
	});

	it('disables the submit button while the mutation is pending', async () => {
		mockPost.mockReturnValue(new Promise(() => {}));

		render(TestWrapper, { props: { component: RegisterPage } });

		const nameInput = page.getByPlaceholder('John Doe');
		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByLabelText('Password');
		const orgInput = page.getByPlaceholder('Acme Inc.');
		const submitButton = page.getByRole('button', { name: 'Create account' });

		await nameInput.fill(validPayload.name);
		await emailInput.fill(validPayload.email);
		await passwordInput.fill(validPayload.password);
		await orgInput.fill(validPayload.org_name);
		await submitButton.click();

		await expect
			.element(page.getByRole('button', { name: /creating account/i }))
			.toBeInTheDocument();
	});

	it('calls the register API and navigates home on success', async () => {
		mockPost.mockResolvedValue({ data: mockAuthResponse });

		render(TestWrapper, { props: { component: RegisterPage } });

		const nameInput = page.getByPlaceholder('John Doe');
		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByLabelText('Password');
		const orgInput = page.getByPlaceholder('Acme Inc.');
		const submitButton = page.getByRole('button', { name: 'Create account' });

		await nameInput.fill(validPayload.name);
		await emailInput.fill(validPayload.email);
		await passwordInput.fill(validPayload.password);
		await orgInput.fill(validPayload.org_name);
		await submitButton.click();

		await vi.waitFor(() => {
			expect(mockPost).toHaveBeenCalledWith('/auth/register', {
				name: validPayload.name,
				email: validPayload.email,
				password: validPayload.password,
				org_name: validPayload.org_name
			});
		});

		await vi.waitFor(() => {
			expect(mockGoto).toHaveBeenCalledWith('/');
		});
	});

	it('shows an error toast on registration failure', async () => {
		mockPost.mockRejectedValue(new Error('Registration failed'));

		render(TestWrapper, { props: { component: RegisterPage } });

		const nameInput = page.getByPlaceholder('John Doe');
		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByLabelText('Password');
		const orgInput = page.getByPlaceholder('Acme Inc.');
		const submitButton = page.getByRole('button', { name: 'Create account' });

		await nameInput.fill(validPayload.name);
		await emailInput.fill(validPayload.email);
		await passwordInput.fill(validPayload.password);
		await orgInput.fill(validPayload.org_name);
		await submitButton.click();

		await vi.waitFor(() => {
			expect(mockToastError).toHaveBeenCalledWith('Registration failed');
		});
	});

	it('provides a link to the login page', async () => {
		render(TestWrapper, { props: { component: RegisterPage } });

		await expect.element(page.getByText('Already have an account?')).toBeInTheDocument();
		const loginLink = page.getByRole('link', { name: 'Log in' });
		await expect.element(loginLink).toBeInTheDocument();
	});
});

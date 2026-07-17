import axios from 'axios';
import { toast } from 'svelte-sonner';

export const api = axios.create({
	baseURL: '/api',
	withCredentials: true,
	headers: { 'Content-Type': 'application/json' }
});

export const axiosErr = (error: Error, fallbackMsg: string) => {
	console.error(error.message);
	if (axios.isAxiosError(error)) toast.error(error.response?.data?.message || fallbackMsg);
	else toast.error(fallbackMsg);
};

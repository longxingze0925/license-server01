import { create } from 'zustand';

interface User {
  id: string;
  email: string;
  name: string;
  role: 'owner' | 'admin' | 'developer' | 'viewer';
  avatar?: string;
  phone?: string;
}

interface Tenant {
  id: string;
  name: string;
  slug: string;
  plan: 'free' | 'pro' | 'enterprise';
  logo?: string;
}

interface AuthState {
  token: string | null;
  user: User | null;
  tenant: Tenant | null;
  isAuthenticated: boolean;
  setAuth: (token: string, user: User, tenant?: Tenant) => void;
  setTenant: (tenant: Tenant) => void;
  updateUser: (user: Partial<User>) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem('token'),
  user: JSON.parse(localStorage.getItem('user') || 'null'),
  tenant: JSON.parse(localStorage.getItem('tenant') || 'null'),
  isAuthenticated: !!localStorage.getItem('token'),
  setAuth: (token, user, tenant) => {
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(user));
    if (tenant) {
      localStorage.setItem('tenant', JSON.stringify(tenant));
    }
    set({ token, user, tenant: tenant || null, isAuthenticated: true });
  },
  setTenant: (tenant) => {
    localStorage.setItem('tenant', JSON.stringify(tenant));
    set({ tenant });
  },
  updateUser: (userData) => {
    set((state) => {
      const newUser = state.user ? { ...state.user, ...userData } : null;
      if (newUser) {
        localStorage.setItem('user', JSON.stringify(newUser));
      }
      return { user: newUser };
    });
  },
  logout: () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('tenant');
    set({ token: null, user: null, tenant: null, isAuthenticated: false });
  },
}));

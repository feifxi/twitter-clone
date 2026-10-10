import { create } from 'zustand';

interface UIState {
  isSignInModalOpen: boolean;
  openSignInModal: () => void;
  closeSignInModal: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  isSignInModalOpen: false,
  openSignInModal: () => set({ isSignInModalOpen: true }),
  closeSignInModal: () => set({ isSignInModalOpen: false }),
}));

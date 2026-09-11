// Example test to verify Jest configuration
describe('Jest Configuration', () => {
  it('should be properly configured', () => {
    expect(true).toBe(true);
  });

  it('should have access to Jest DOM matchers', () => {
    const element = document.createElement('div');
    document.body.appendChild(element); // Add to document first
    expect(element).toBeInTheDocument();
    document.body.removeChild(element); // Clean up
  });

  it('should support async tests', async () => {
    const promise = Promise.resolve(42);
    const result = await promise;
    expect(result).toBe(42);
  });
});

// Example Zustand store test
import { create } from 'zustand';

interface TestStore {
  count: number;
  increment: () => void;
}

const useTestStore = create<TestStore>((set) => ({
  count: 0,
  increment: () => set((state) => ({ count: state.count + 1 })),
}));

describe('Zustand Integration', () => {
  it('should create a store successfully', () => {
    const { count, increment } = useTestStore.getState();
    expect(count).toBe(0);
    
    increment();
    
    const updatedCount = useTestStore.getState().count;
    expect(updatedCount).toBe(1);
  });
});
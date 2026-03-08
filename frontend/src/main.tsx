import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

// Log all fetch requests
const originalFetch = window.fetch;
window.fetch = async (input, init) => {
  // Convert input to URL object to handle relative paths
  const url = new URL(input instanceof Request ? input.url : input, window.location.origin);
  
  // Skip logging for vite's internal requests
  if (url.pathname.startsWith('/@')) {
    return originalFetch(input, init);
  }

  const startTime = performance.now();
  const request = new Request(input, {
    ...init,
    headers: {
      ...init?.headers,
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    }
  });
  
  console.log('Fetch initiated:', {
    url: request.url,
    method: request.method,
    headers: Object.fromEntries(request.headers.entries())
  });

  try {
    const response = await originalFetch(request);
    const duration = (performance.now() - startTime).toFixed(0);
    const text = await response.clone().text();
    
    console.log(`Fetch completed in ${duration}ms:`, {
      url: response.url,
      status: response.status,
      headers: Object.fromEntries(response.headers.entries()),
      redirected: response.redirected,
      body: text
    });

    // If we got HTML when expecting JSON, throw error
    if (response.headers.get('content-type')?.includes('text/html') && 
        request.headers.get('accept')?.includes('application/json')) {
      throw new Error(`Received HTML when expecting JSON from ${response.url}`);
    }

    return response;
  } catch (error) {
    console.error('Fetch failed:', error);
    throw error;
  }
};

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)

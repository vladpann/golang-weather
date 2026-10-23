// Frontend authentication middleware
// This script checks if user is authenticated and redirects to sign-in if not

const AUTH_CHECK_URL = 'http://localhost/api/v1/auth/check';
const SIGN_IN_URL = 'sign-in.html';

// Pages that require authentication
const PROTECTED_PAGES = ['index.html', 'search-results.html'];

// Check if current page is protected
function isProtectedPage() {
  const currentPage = window.location.pathname.split('/').pop();
  return PROTECTED_PAGES.includes(currentPage);
}

// Check authentication status
async function checkAuth() {
  // Only check auth on protected pages
  if (!isProtectedPage()) {
    return true;
  }

  try {
    const response = await fetch(AUTH_CHECK_URL, {
      method: 'GET',
      credentials: 'include',
    });

    if (response.ok) {
      // User is authenticated
      return true;
    } else {
      // User is not authenticated, redirect to sign-in
      window.location.href = SIGN_IN_URL;
      return false;
    }
  } catch (error) {
    console.error('Auth check failed:', error);
    // On network error, redirect to sign-in
    window.location.href = SIGN_IN_URL;
    return false;
  }
}

// Run auth check when page loads
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', checkAuth);
} else {
  checkAuth();
}

// Export for use in other scripts
window.authMiddleware = {
  checkAuth,
  isProtectedPage,
};

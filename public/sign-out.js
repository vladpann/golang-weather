// Sign-out functionality
const SIGN_OUT_URL = 'http://localhost/api/v1/auth/sign-out';
const SIGN_IN_PAGE = 'sign-in.html';

async function signOut() {
  try {
    const response = await fetch(SIGN_OUT_URL, {
      method: 'POST',
      credentials: 'include',
    });

    if (response.ok || response.status === 204) {
      // Sign-out successful, redirect to sign-in page
      window.location.href = SIGN_IN_PAGE;
    } else {
      console.error('Sign-out failed:', response.status);
      // Even if sign-out fails, redirect to sign-in
      window.location.href = SIGN_IN_PAGE;
    }
  } catch (error) {
    console.error('Sign-out error:', error);
    // On error, still redirect to sign-in
    window.location.href = SIGN_IN_PAGE;
  }
}

// Attach sign-out handler when DOM is ready
function attachSignOutHandler() {
  const signOutBtn = document.getElementById('sign-out-btn');
  if (signOutBtn) {
    signOutBtn.addEventListener('click', (e) => {
      e.preventDefault();
      signOut();
    });
  }
}

// Run when page loads
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', attachSignOutHandler);
} else {
  attachSignOutHandler();
}

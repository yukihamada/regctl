// Demo tab functionality
function showDemo(demoType) {
    // Hide all demo contents
    const contents = document.querySelectorAll('.demo-content');
    contents.forEach(content => {
        content.classList.remove('active');
    });
    
    // Remove active class from all tabs
    const tabs = document.querySelectorAll('.tab');
    tabs.forEach(tab => {
        tab.classList.remove('active');
    });
    
    // Show selected demo content
    const selectedContent = document.getElementById(`demo-${demoType}`);
    if (selectedContent) {
        selectedContent.classList.add('active');
    }
    
    // Add active class to clicked tab
    const clickedTab = event.target;
    clickedTab.classList.add('active');
}

// Copy to clipboard functionality
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        // Show temporary success feedback
        const button = event.target;
        const originalText = button.textContent;
        button.textContent = '✅';
        button.style.background = '#10b981';
        
        setTimeout(() => {
            button.textContent = originalText;
            button.style.background = '';
        }, 2000);
    }).catch(err => {
        console.error('Failed to copy: ', err);
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = text;
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand('copy');
        document.body.removeChild(textArea);
        
        // Show feedback
        const button = event.target;
        const originalText = button.textContent;
        button.textContent = '✅';
        setTimeout(() => {
            button.textContent = originalText;
        }, 2000);
    });
}

// Update quick command with custom domain
function updateQuickCommand() {
    const domainInput = document.getElementById('domain-input');
    const quickCommand = document.getElementById('quick-command');
    const domain = domainInput.value || 'mobility360.jp';
    
    const newCommand = `curl -fsSL https://regctl.com/quick.sh | bash -s ${domain}`;
    const commandSpan = quickCommand.querySelector('.command');
    const copyBtn = quickCommand.querySelector('.copy-btn');
    
    if (commandSpan) {
        commandSpan.textContent = newCommand;
    }
    
    if (copyBtn) {
        copyBtn.setAttribute('onclick', `copyToClipboard('${newCommand}')`);
    }
}

// Smooth scrolling for anchor links
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    anchor.addEventListener('click', function (e) {
        e.preventDefault();
        const target = document.querySelector(this.getAttribute('href'));
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        }
    });
});

// A/B Test Analytics
function trackABTestEvent(eventName, additionalData = {}) {
    const abTestMeta = document.querySelector('meta[name="ab-test-variants"]');
    if (abTestMeta) {
        const variants = abTestMeta.getAttribute('content');
        const userId = abTestMeta.getAttribute('data-user');
        
        // Send analytics event (Google Analytics 4 or custom analytics)
        if (typeof gtag !== 'undefined') {
            gtag('event', eventName, {
                'custom_parameter_ab_variants': variants,
                'custom_parameter_user_id': userId,
                ...additionalData
            });
        }
        
        // Also log for debugging
        console.log(`A/B Test Event: ${eventName}`, {
            variants,
            userId,
            ...additionalData
        });
    }
}

// Enhanced copy function with A/B test tracking
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        // Track copy event for A/B testing
        trackABTestEvent('copy_command', { command_type: 'install' });
        
        // Show temporary success feedback
        const button = event.target;
        const originalText = button.textContent;
        button.textContent = '✅';
        button.style.background = '#10b981';
        
        setTimeout(() => {
            button.textContent = originalText;
            button.style.background = '';
        }, 2000);
    }).catch(err => {
        console.error('Failed to copy: ', err);
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = text;
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand('copy');
        document.body.removeChild(textArea);
        
        // Show feedback
        const button = event.target;
        const originalText = button.textContent;
        button.textContent = '✅';
        setTimeout(() => {
            button.textContent = originalText;
        }, 2000);
    });
}

// DOMContentLoaded event handler
document.addEventListener('DOMContentLoaded', () => {
    // Check A/B test variants and initialize accordingly
    const abTestMeta = document.querySelector('meta[name="ab-test-variants"]');
    let defaultDemo = 'cli'; // default
    
    if (abTestMeta) {
        const variants = abTestMeta.getAttribute('content');
        const variantPairs = variants.split(',');
        
        variantPairs.forEach(pair => {
            const [testName, variantIndex] = pair.split(':');
            if (testName === 'demoDefault') {
                const demoOptions = ['cli', 'claude', 'api'];
                defaultDemo = demoOptions[parseInt(variantIndex)] || 'cli';
            }
        });
        
        // Track A/B test view
        trackABTestEvent('page_view');
    }
    
    // Initialize demo tab based on A/B test
    const targetTab = document.querySelector(`.tab[onclick="showDemo('${defaultDemo}')"]`);
    const targetDemo = document.getElementById(`demo-${defaultDemo}`);
    
    if (targetTab && targetDemo) {
        // Remove any existing active classes
        document.querySelectorAll('.tab').forEach(tab => tab.classList.remove('active'));
        document.querySelectorAll('.demo-content').forEach(demo => demo.classList.remove('active'));
        
        // Set the A/B test determined tab as active
        targetTab.classList.add('active');
        targetDemo.classList.add('active');
    } else {
        // Fallback to first tab
        const firstTab = document.querySelector('.tab');
        const firstDemo = document.querySelector('.demo-content');
        
        if (firstTab && firstDemo) {
            firstTab.classList.add('active');
            firstDemo.classList.add('active');
        }
    }
    
    // Track button clicks for A/B testing
    document.querySelectorAll('.btn-primary').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const buttonText = e.target.textContent.trim();
            trackABTestEvent('cta_click', { button_text: buttonText });
        });
    });
    
    // Track pricing card interactions
    document.querySelectorAll('.pricing-card').forEach(card => {
        card.addEventListener('click', (e) => {
            const planName = card.querySelector('h3')?.textContent;
            trackABTestEvent('pricing_click', { plan_name: planName });
        });
    });
    
    // Add intersection observer for fade-in animations
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -100px 0px'
    };
    
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, observerOptions);
    
    // Observe animated elements
    document.querySelectorAll('.feature-card, .pricing-card, .case-card, .install-option').forEach(el => {
        el.style.opacity = '0';
        el.style.transform = 'translateY(20px)';
        el.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
        observer.observe(el);
    });
    
    // Add scroll effect to header
    window.addEventListener('scroll', function() {
        const header = document.querySelector('header');
        const scrolled = window.pageYOffset;
        
        if (scrolled > 50) {
            header.style.backdropFilter = 'blur(20px)';
            header.style.background = 'rgba(255, 255, 255, 0.95)';
        } else {
            header.style.backdropFilter = 'blur(10px)';
            header.style.background = 'rgba(255, 255, 255, 0.9)';
        }
    });
});

// Mobile menu toggle (if needed in future)
function toggleMobileMenu() {
    const navLinks = document.querySelector('.nav-links');
    navLinks.classList.toggle('mobile-active');
}

// Theme toggle (for future dark mode support)
function toggleTheme() {
    document.body.classList.toggle('dark-theme');
    localStorage.setItem('theme', document.body.classList.contains('dark-theme') ? 'dark' : 'light');
}

// Check for saved theme preference
const savedTheme = localStorage.getItem('theme');
if (savedTheme === 'dark') {
    document.body.classList.add('dark-theme');
}
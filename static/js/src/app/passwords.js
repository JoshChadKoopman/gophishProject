import zxcvbn from 'zxcvbn';

const PasswordPolicy = [
    { id: 'policy-length',  test: v => v.length >= 10 },
    { id: 'policy-upper',   test: v => /[A-Z]/.test(v) },
    { id: 'policy-lower',   test: v => /[a-z]/.test(v) },
    { id: 'policy-digit',   test: v => /[0-9]/.test(v) },
    { id: 'policy-special', test: v => /[^A-Za-z0-9]/.test(v) }
];

function updatePolicyChecklist(value) {
    const list = document.getElementById('password-policy-checklist');
    if (!list) return;
    list.style.display = value ? '' : 'none';
    PasswordPolicy.forEach(function(rule) {
        const li = document.getElementById(rule.id);
        if (!li) return;
        const icon = li.querySelector('i');
        if (rule.test(value)) {
            icon.className = 'fa fa-check text-success';
        } else {
            icon.className = 'fa fa-times text-danger';
        }
    });
}

const StrengthMapping = {
    0: {
        class: 'danger',
        width: '10%',
        status: 'Very Weak'
    },
    1: {
        class: 'danger',
        width: '25%',
        status: 'Very Weak'
    },
    2: {
        class: 'warning',
        width: '50%',
        status: 'Weak'
    },
    3: {
        class: 'success',
        width: '75%',
        status: 'Good'
    },
    4: {
        class: 'success',
        width: '100%',
        status: 'Very Good'
    }
}

const Progress = document.getElementById("password-strength-container")
const ProgressBar = document.getElementById("password-strength-bar")
const StrengthDescription = document.getElementById("password-strength-description")

const updatePasswordStrength = (e) => {
    const candidate = e.target.value
    // If there is no password, clear out the progress bar
    if (!candidate) {
        ProgressBar.style.width = 0
        StrengthDescription.textContent = ""
        Progress.classList.add("hidden")
        return
    }
    const score = zxcvbn(candidate).score
    const evaluation = StrengthMapping[score]
    // Update the progress bar
    ProgressBar.classList = `progress-bar progress-bar-${evaluation.class}`
    ProgressBar.style.width = evaluation.width
    StrengthDescription.textContent = evaluation.status
    StrengthDescription.classList = `text-${evaluation.class}`
    Progress.classList.remove("hidden")
    updatePolicyChecklist(candidate)
}

document.getElementById("password").addEventListener("input", updatePasswordStrength)
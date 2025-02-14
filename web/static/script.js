function toggleCheck(uuid, enabled) {
    fetch('/api/toggle-check', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: `uuid=${uuid}&enabled=${enabled}`
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
    })
    .catch(error => {
        console.error('Error:', error);
        // Revert the toggle if there was an error
        const checkbox = document.querySelector(`input[data-uuid="${uuid}"]`);
        if (checkbox) {
            checkbox.checked = !enabled;
        }
    });
} 
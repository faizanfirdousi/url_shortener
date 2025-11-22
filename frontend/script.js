document.getElementById('shorten-form').addEventListener('submit', async (event) => {
    event.preventDefault();

    const url = document.getElementById('url-input').value;
    const resultDiv = document.getElementById('result');
// hello
    try {
        const response = await fetch('/url', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ url }),
        });

        const data = await response.json();

        if (response.ok) {
            const shortenedURL = `${window.location.origin}/${data.alias}`;
            resultDiv.innerHTML = `
                <p>Shortened URL:</p>
                <a href="${shortenedURL}" target="_blank">${shortenedURL}</a>
            `;
        } else {
            resultDiv.innerHTML = `<p>Error: ${data.error}</p>`;
        }
    } catch (error) {
        resultDiv.innerHTML = `<p>Error: Could not connect to the server.</p>`;
        console.error('Error:', error);
    }
});
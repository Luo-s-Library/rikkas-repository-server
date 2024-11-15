updateTable()
function linkFile() {
    const fileInput = document.getElementById('fileInput');
    const file = fileInput.files[0];
    const formData = new FormData();
    formData.append('file', file);

    fetch('/api/link', {
        method: 'POST',
        body: formData
    }).then(response => response.text())
        .then(result => {
        updateTable()
        })
        .catch(err => console.error('Error: ', error));
}
function uploadFile() {
    const fileInput = document.getElementById('fileInput');
    const file = fileInput.files[0];
    const formData = new FormData();
    formData.append('file', file);

    fetch('/api/upload', {
        method: 'POST',
        body: formData
    }).then(response => response.text())
        .then(result => {
        updateTable()
        })
        .catch(error => console.error('Error:', error));
}
function makeTD(text) {
    const cell = document.createElement('td');
    cell.classList.add('styled-table')
    cell.style.fontSize = "24px"
    cell.textContent = text
    return cell
}
function makeCoverImage(title) {
    const td = document.createElement('td')

    const img = document.createElement('img')
    img.src = '/api/getcover?title='+title
    img.style.width = "50px"
    img.style.height = "auto"
    td.appendChild(img)

    return td
}
function createDownloadButton(title) {
    const td = document.createElement('td')

    // Create button element
    const button = document.createElement('button');
    button.innerText = `Download ${title}`; // Set button text

    // Style the button
    button.style.padding = '10px 20px';
    button.style.border = 'none';
    button.style.backgroundColor = '#4CAF50'; // Green background
    button.style.color = 'white'; // White text
    button.style.cursor = 'pointer';
    button.style.borderRadius = '5px';

    // Add hover effect
    button.onmouseover = function() {
        button.style.backgroundColor = '#45a049'; // Darker green on hover
    };
    button.onmouseout = function() {
        button.style.backgroundColor = '#4CAF50'; // Original green
    };

    // Set the button click event
    button.onclick = function() {
        // Create a temporary anchor element for download
        const a = document.createElement('a');
        a.href = `/api/getbook?title=${encodeURIComponent(title)}`; // Set download URL
        a.download = title; // Set the suggested file name
        document.body.appendChild(a); // Append to the body (needed for Firefox)
        a.click(); // Trigger the download
        document.body.removeChild(a); // Remove the anchor element
    };

    td.appendChild(button)

    return td
}
function createButtonInCell(title) {
    // Create td element
    const td = document.createElement('td');

    // Create button element
    const button = document.createElement('button');
    button.innerText = title; // Set button text
    
    // Style the button
    button.style.padding = '10px 20px';
    button.style.border = 'none';
    button.style.backgroundColor = '#4CAF50'; // Green background
    button.style.color = 'white'; // White text
    button.style.cursor = 'pointer';
    button.style.borderRadius = '5px';

    // Add hover effect
    button.onmouseover = function() {
        button.style.backgroundColor = '#45a049'; // Darker green on hover
    };
    button.onmouseout = function() {
        button.style.backgroundColor = '#4CAF50'; // Original green
    };

    // Set the button click event
    button.onclick = function() {
        fetch(`/api/generatemp3?title=${encodeURIComponent(title)}`)
            .then(response => {
                if (response.ok) {
                    console.log("MP3 generation triggered.");
                } else {
                    console.error("Failed to generate MP3.");
                }
                updateTable()
            })
            .catch(error => console.error('Error:', error));
    };

    // Append the button to the td
    td.appendChild(button);

    // Return the td element to be inserted into the table
    return td;
}
function updateTable() {
    fetch('/api/getbooklist')
    .then(response => response.json())
    .then(data => {
        const tableBody = document.querySelector('#dataTable tbody');
        
        tableBody.innerHTML = ''

        if (data.books === null)
            return
        data.books.forEach(item => {
            const row = document.createElement('tr');
            row.classList.add("styled-table")

            const indexCell = makeCoverImage(item.title)
            row.appendChild(indexCell);
            
            const titleCell = makeTD(item.title);
            row.appendChild(titleCell);

            const downloadCell = createDownloadButton(item.title);
            row.appendChild(downloadCell);

            tableBody.appendChild(row);
        })
    })
    .catch(err => console.error('Error fetching data:', err));
}
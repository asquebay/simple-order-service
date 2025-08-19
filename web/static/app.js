document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('id-form');
    const input = document.getElementById('order-uid-input');
    const detailsContainer = document.getElementById('order-details');
    const errorContainer = document.getElementById('error-message');

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const uid = input.value.trim();

        // очищаем предыдущие результаты
        detailsContainer.textContent = '';
        errorContainer.textContent = '';
        detailsContainer.style.display = 'none';

        if (!uid) {
            errorContainer.textContent = 'Please enter an Order UID';
            return;
        }

        // формируем URL для запроса к API
        const apiUrl = `/order/${uid}`;

        try {
            const response = await fetch(apiUrl);

            // если ответ не ok (статус не 200-299), обрабатываем как ошибку
            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || `Error: ${response.status}`);
            }

            const orderData = await response.json();
            displayOrder(orderData);

        } catch (error) {
            console.error('API error:', error);
            displayError(error.message);
        }
    });

    function displayOrder(data) {
        // используем JSON.stringify для красивого форматирования JSON
        detailsContainer.textContent = JSON.stringify(data, null, 2);
        detailsContainer.style.display = 'block';
    }

    function displayError(message) {
        errorContainer.textContent = message;
    }
});

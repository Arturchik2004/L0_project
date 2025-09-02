document.addEventListener('DOMContentLoaded', () => {
   const searchButton = document.getElementById('searchButton');
   const orderUidInput = document.getElementById('orderUidInput');
   const resultContainer = document.getElementById('resultContainer');
   const errorContainer = document.getElementById('errorContainer');
   const loader = document.getElementById('loader');
   const recentOrdersContainer = document.getElementById('recentOrdersContainer');
   const recentTitle = document.getElementById('recent-title');

   const showLoader = () => loader.classList.remove('hidden');
   const hideLoader = () => loader.classList.add('hidden');
   const showError = (message) => {
       errorContainer.textContent = message;
       errorContainer.classList.remove('hidden');
   };
   const hideError = () => errorContainer.classList.add('hidden');
   const clearResults = () => {
       resultContainer.innerHTML = '';
       resultContainer.classList.add('hidden');
   }

   async function fetchOrder() {
       const orderUid = orderUidInput.value.trim();
       if (!orderUid) return;

       showLoader();
       searchButton.disabled = true;
       hideError();
       clearResults();
       recentOrdersContainer.classList.add('hidden');
       recentTitle.classList.add('hidden');

       try {
           const response = await fetch(`/api/order/${orderUid}`);
           if (!response.ok) {
               const errorText = await response.text();
               throw new Error(`Заказ не найден (Статус: ${response.status}). ${errorText}`);
           }
           const data = await response.json();
           displayOrder(data);
       } catch (error) {
           showError(error.message);
       } finally {
           hideLoader();
           searchButton.disabled = orderUidInput.value.trim() === '';
       }
   }

   function displayOrder(order) {
       const createGridItem = (label, value) => {
           const item = document.createElement('div');
           item.className = 'grid-item';
           const strong = document.createElement('strong');
           strong.textContent = `${label}: `;
           item.appendChild(strong);
           item.append(value || 'N/A');
           return item;
       };

       const createCard = (title, ...children) => {
           const card = document.createElement('div');
           card.className = 'card';
           const h2 = document.createElement('h2');
           h2.textContent = title;
           card.appendChild(h2);
           const grid = document.createElement('div');
           grid.className = 'grid';
           children.forEach(child => grid.appendChild(child));
           card.appendChild(grid);
           return card;
       };

       const generalInfoCard = createCard('Общая информация',
           createGridItem('UID Заказа', order.order_uid),
           createGridItem('Трек-номер', order.track_number),
           createGridItem('ID Покупателя', order.customer_id),
           createGridItem('Дата создания', new Date(order.date_created).toLocaleString()),
           createGridItem('Служба доставки', order.delivery_service),
           createGridItem('Entry', order.entry)
       );

       const deliveryCard = createCard('Доставка',
           createGridItem('Имя', order.delivery.name),
           createGridItem('Телефон', order.delivery.phone),
           createGridItem('Email', order.delivery.email),
           createGridItem('Город', order.delivery.city),
           createGridItem('Индекс', order.delivery.zip),
           createGridItem('Адрес', order.delivery.address)
       );

       const paymentCard = createCard('Оплата',
           createGridItem('Транзакция', order.payment.transaction),
           createGridItem('Сумма', `${order.payment.amount} ${order.payment.currency}`),
           createGridItem('Стоимость доставки', order.payment.delivery_cost),
           createGridItem('Всего товаров', order.payment.goods_total),
           createGridItem('Банк', order.payment.bank),
           createGridItem('Провайдер', order.payment.provider)
       );

       const itemsCard = document.createElement('div');
       itemsCard.className = 'card';
       const h2 = document.createElement('h2');
       h2.textContent = 'Состав заказа';
       itemsCard.appendChild(h2);
       order.items.forEach(item => {
           const itemCard = document.createElement('div');
           itemCard.className = 'item-card';
           const grid = document.createElement('div');
           grid.className = 'grid';
           grid.append(
               createGridItem('Название', item.name),
               createGridItem('Бренд', item.brand),
               createGridItem('Артикул (nm_id)', item.nm_id),
               createGridItem('Цена', item.total_price),
               createGridItem('Скидка', `${item.sale}%`)
           );
           itemCard.appendChild(grid);
           itemsCard.appendChild(itemCard);
       });

       resultContainer.append(generalInfoCard, deliveryCard, paymentCard, itemsCard);
       resultContainer.classList.remove('hidden');
   }
   
   function displayOrderList(orders) {
       recentOrdersContainer.innerHTML = '';
       orders.forEach(order => {
           const card = document.createElement('div');
           card.className = 'card recent-order-card';
           card.innerHTML = `
               <div class="grid">
                   <div class="grid-item"><strong>UID:</strong> ${order.order_uid}</div>
                   <div class="grid-item"><strong>Покупатель:</strong> ${order.customer_id}</div>
                   <div class="grid-item"><strong>Дата:</strong> ${new Date(order.date_created).toLocaleString()}</div>
               </div>
           `;
           
           card.addEventListener('click', () => {
               orderUidInput.value = order.order_uid;
               fetchOrder();
               window.scrollTo(0, 0);
           });

           recentOrdersContainer.appendChild(card);
       });
   }

   async function fetchAndDisplayRecentOrders() {
       try {
           const response = await fetch('/api/orders/recent');
           if (!response.ok) throw new Error('Ошибка загрузки');
           const orders = await response.json();
           
           if (orders && orders.length > 0) {
               recentTitle.classList.remove('hidden');
               displayOrderList(orders);
           }
       } catch (error) {
           console.error("Не удалось загрузить последние заказы:", error);
       }
   }

   fetchAndDisplayRecentOrders();

   searchButton.addEventListener('click', fetchOrder);
   orderUidInput.addEventListener('keypress', (event) => {
       if (event.key === 'Enter') {
           fetchOrder();
       }
   });
   orderUidInput.addEventListener('input', () => {
       searchButton.disabled = orderUidInput.value.trim() === '';
   });
});
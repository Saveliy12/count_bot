const money = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "RUB",
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
});

const total = document.querySelector("#total");
const count = document.querySelector("#count");

async function loadSummary() {
  try {
    const response = await fetch("/api/v1/expenses?limit=1");
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const data = await response.json();
    total.textContent = money.format(Number(data.total));
    count.textContent = data.count === 1 ? "1 трата" : `${data.count} трат`;
  } catch {
    total.textContent = "Нет связи";
    count.textContent = "Не удалось получить данные из API";
  }
}

loadSummary();

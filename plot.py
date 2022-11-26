import matplotlib.pyplot as plt
import csv

x = []
y = []

with open('results.csv','r') as csvfile:
    lines = csv.reader(csvfile, delimiter=',')
    for row in lines:
        x.append(row[0])
        y.append(row[1])

plt.plot(x, y, color = 'g', linestyle = 'dashed',
         marker = 'o',label = "Time vs Thread")

plt.xticks(rotation = 25)
plt.xlabel('Number of Thread')
plt.ylabel('Time (s)')
plt.title('Dist Graph', fontsize = 20)
plt.gca().invert_yaxis()
plt.grid()
plt.legend()
plt.show()
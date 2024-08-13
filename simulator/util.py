import numpy as np

def remove_outliers_iqr(data, threshold=1.5):
    q1 = np.percentile(data, 15)
    q3 = np.percentile(data, 85)
    iqr = q3 - q1
    lower_bound = q1 - threshold * iqr
    upper_bound = q3 + threshold * iqr
    outliers_removed = data[(data >= lower_bound) & (data <= upper_bound)]
    print(len(outliers_removed),"d")
    return outliers_removed

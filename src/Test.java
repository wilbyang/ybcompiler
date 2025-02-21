// 文件名: Main.java

class Animal {
    public void makeSound() {
        System.out.println("Generic sound");
    }
}

class Dog extends Animal {
    @Override
    public void makeSound() {
        System.out.println("Woof");
    }
}

public class Test {
    public static void main(String[] args) {
        Animal animal = new Dog(); // 引用类型是 Animal，实际类型是 Dog
        animal.makeSound();       // 输出 "Woof"
    }
}